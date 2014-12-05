package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"path"
	"time"

	deb ".."
)

// ClientBuilder is a DebianBuilder that defers build operation to
// another DebianBuilder through a unix socket (should be on the same
// system)
type ClientBuilder struct {
	conn *rpc.Client
}

func NewClientBuilder(network, addr string) (*ClientBuilder, error) {
	var err error
	res := &ClientBuilder{}
	res.conn, err = rpc.DialHTTP(network, addr)
	return res, err
}

// syncs the output of the to the given writer, errChan is here to
// report that all the data was consumed.
func (s *SyncOutputResults) sync(output io.Writer, errChan chan error) (*bytes.Buffer, error) {
	// we connect to the given address, and write output to both
	// writter and buffer
	conn, err := net.Dial(s.Network, s.Address)
	if err != nil {
		return nil, err
	}
	logData := &bytes.Buffer{}
	go func() {
		var logDest io.Writer = logData
		if output != nil {
			logDest = io.MultiWriter(logData, output)
		}

		_, err := io.Copy(logDest, conn)
		defer func() { errChan <- err }()
		if err != nil {
			return
		}
		err = conn.Close()
	}()

	return logData, nil
}

func (c *ClientBuilder) initSynchronization(output io.Writer) (SyncOutputID, *bytes.Buffer, chan error, error) {
	initArgs := NoValue{}
	syncOut := SyncOutputResults{}
	if err := c.conn.Call("RpcBuilder.InitSync", initArgs, &syncOut); err != nil {
		return 0, nil, nil, err
	}
	errChan := make(chan error)
	logData, err := syncOut.sync(output, errChan)
	if err != nil {
		return 0, nil, nil, err
	}
	return syncOut.ID, logData, errChan, nil
}

func (c *ClientBuilder) BuildPackage(args BuildArguments, output io.Writer) (*BuildResult, error) {
	id, logData, errChan, err := c.initSynchronization(output)
	if err != nil {
		return nil, err
	}
	rArgs := RpcBuildArguments{
		ID:   id,
		Args: args,
	}
	res := &BuildResult{}
	if err = c.conn.Call("RpcBuilder.Build", rArgs, res); err != nil {
		return nil, err
	}
	err = <-errChan
	res.BuildLog = Log(logData.String())

	return res, err
}

func (c *ClientBuilder) InitDistribution(d deb.Distribution, a deb.Architecture, output io.Writer) error {
	id, _, errChan, err := c.initSynchronization(output)
	if err != nil {
		return err
	}

	args := CreateArgs{
		ID:   id,
		Dist: d,
		Arch: a,
	}
	if err = c.conn.Call("RpcBuilder.Create", args, nil); err != nil {
		return err
	}
	err = <-errChan
	return err
}

func (c *ClientBuilder) RemoveDistribution(d deb.Distribution, a deb.Architecture) error {
	return c.conn.Call("RpcBuilder.Remove", RpcDistAndArchArgs{Dist: d, Arch: a}, &NoValue{})
}

func (c *ClientBuilder) UpdateDistribution(d deb.Distribution, a deb.Architecture) error {
	return c.conn.Call("RpcBuilder.Update", RpcDistAndArchArgs{Dist: d, Arch: a}, &NoValue{})
}

func (c *ClientBuilder) AvailableDistributions() []deb.Distribution {
	res := DistributionList{}
	if err := c.conn.Call("RpcBuilder.AvailableDistributions", NoValue{}, &res); err != nil {
		panic(err)
	}
	return res.Dists
}

func (c *ClientBuilder) AvailableArchitectures(d deb.Distribution) []deb.Architecture {
	res := ArchitectureList{}
	if err := c.conn.Call("RpcBuilder.AvailableArchitectures", d, &res); err != nil {
		panic(err)
	}
	return res.Archs
}

type SyncOutputID uint64

type syncOutput struct {
	id      SyncOutputID
	w       io.WriteCloser
	timeout *time.Time
	file    string
}

func (s syncOutput) Close() {
	if s.w != nil {
		s.w.Close()
	}
	if len(s.file) != 0 {
		os.Remove(s.file)
	}
}

type SyncOutputResults struct {
	//the address of the unix socket that will be use to send the
	//build output
	Network, Address string
	//the ID of the current synchronized output
	ID SyncOutputID
}

type RpcBuilder struct {
	actualBuilder DebianBuilder

	ticker                             chan bool
	generator                          chan SyncOutputID
	timeoutClearer, timeouter, remover chan SyncOutputID
	syncOutputs                        map[SyncOutputID]syncOutput
}

func (b *RpcBuilder) manageSyncID() {
	newId := SyncOutputID(0)

	b.ticker = make(chan bool)

	go func() {
		time.Sleep(10 * time.Second)
		b.ticker <- true
	}()
	i := 0
	for {
		select {
		case b.generator <- newId:
			newId = newId + 1
		case toRemove := <-b.remover:
			b.syncOutputs[toRemove].Close()
			delete(b.syncOutputs, toRemove)
		case id := <-b.timeouter:
			s, ok := b.syncOutputs[id]
			if ok == true {
				time := time.Now().Add(100 * time.Second)
				s.timeout = &time
			}
		case id := <-b.timeoutClearer:
			s, ok := b.syncOutputs[id]
			if ok == true {
				s.timeout = nil
			}
		case _ = <-b.ticker:
			for k, v := range b.syncOutputs {
				if v.timeout == nil {
					continue
				}
				if time.Now().After(*v.timeout) {
					v.Close()
					delete(b.syncOutputs, k)
				}
			}
		}
		i = i + 1
	}
}

func (b *RpcBuilder) waitForSyncOutputConnection(l net.Listener, id SyncOutputID) {
	b.timeouter <- id
	defer l.Close()
	conn, err := l.Accept()
	if err != nil {
		//todo: log
		return
	}
	b.timeoutClearer <- id
	s, ok := b.syncOutputs[id]
	if ok == false {
		//todo: loh
		return
	}
	s.w = conn
	b.syncOutputs[id] = s
	b.timeouter <- id
}

type NoValue struct{}

func (b *RpcBuilder) InitSync(args NoValue, res *SyncOutputResults) error {
	tmpDir, err := ioutil.TempDir("", "sync-socket")
	if err != nil {
		return err
	}
	tmpFile := path.Join(tmpDir, "sync.sock")
	l, err := net.Listen("unix", tmpFile)
	if err != nil {
		return err
	}

	id := <-b.generator
	s := syncOutput{
		id:   id,
		file: tmpFile,
	}
	b.syncOutputs[id] = s
	res.ID = id
	res.Network = "unix"
	res.Address = tmpFile
	go b.waitForSyncOutputConnection(l, id)
	return nil
}

type RpcBuildArguments struct {
	ID   SyncOutputID
	Args BuildArguments
}

func (b *RpcBuilder) Build(args RpcBuildArguments, res *BuildResult) error {
	b.timeoutClearer <- args.ID
	s, ok := b.syncOutputs[args.ID]
	if ok == false {
		return fmt.Errorf("No output synchronization %d available", args.ID)
	}
	defer func() { b.remover <- args.ID }()
	writer := s.w
	if writer == nil {
		return fmt.Errorf("Client is not connected to synchronization output %d", args.ID)
	}

	actualRes, err := b.actualBuilder.BuildPackage(args.Args, s.w)
	if err != nil {
		return err
	}
	// do not re-send build log over network
	if actualRes != nil {
		actualRes.BuildLog = Log("")
		*res = *actualRes
	}
	return nil
}

type CreateArgs struct {
	ID   SyncOutputID
	Dist deb.Distribution
	Arch deb.Architecture
}

func (b *RpcBuilder) Create(args CreateArgs, res *NoValue) error {
	b.timeoutClearer <- args.ID
	s, ok := b.syncOutputs[args.ID]
	if ok == false {
		return fmt.Errorf("No output synchronization %d available", args.ID)
	}
	defer func() { b.remover <- args.ID }()
	writer := s.w
	if writer == nil {
		return fmt.Errorf("Client is not connected to synchronization output %d", args.ID)
	}

	return b.actualBuilder.InitDistribution(args.Dist, args.Arch, writer)
}

type RpcDistAndArchArgs struct {
	Dist deb.Distribution
	Arch deb.Architecture
}

func (b *RpcBuilder) Remove(args RpcDistAndArchArgs, res *NoValue) error {
	return b.actualBuilder.RemoveDistribution(args.Dist, args.Arch)
}

func (b *RpcBuilder) Update(args RpcDistAndArchArgs, res *NoValue) error {
	return b.actualBuilder.UpdateDistribution(args.Dist, args.Arch)
}

type DistributionList struct {
	Dists []deb.Distribution
}

func (b *RpcBuilder) AvailableDistributions(arg NoValue, res *DistributionList) error {
	res.Dists = b.actualBuilder.AvailableDistributions()
	return nil
}

type ArchitectureList struct {
	Archs []deb.Architecture
}

func (b *RpcBuilder) AvailableArchitectures(d deb.Distribution, res *ArchitectureList) error {
	res.Archs = b.actualBuilder.AvailableArchitectures(d)
	return nil
}

type RpcBuilderServer struct {
	b                *RpcBuilder
	network, address string
	errChan          chan error
}

func NewRcpBuilderServer(builder DebianBuilder, network, address string) *RpcBuilderServer {
	return &RpcBuilderServer{
		b: &RpcBuilder{
			actualBuilder: builder,
		},
		network: network,
		address: address,
		errChan: make(chan error),
	}
}

func (s *RpcBuilderServer) Serve() {
	if s.b.generator != nil {
		s.errChan <- fmt.Errorf("Already started")
	}
	if err := rpc.Register(s.b); err != nil {
		s.errChan <- err
		return
	}
	rpc.HandleHTTP()
	l, err := net.Listen(s.network, s.address)
	if err != nil {
		s.errChan <- err
		return
	}

	s.b.generator = make(chan SyncOutputID)
	s.b.timeouter = make(chan SyncOutputID)
	s.b.timeoutClearer = make(chan SyncOutputID)
	s.b.remover = make(chan SyncOutputID)
	s.b.syncOutputs = make(map[SyncOutputID]syncOutput)

	go s.b.manageSyncID()
	s.errChan <- nil
	http.Serve(l, nil)
}

func (s *RpcBuilderServer) WaitEstablished() error {
	return <-s.errChan
}
