package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
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
	return fmt.Errorf("Client builder are not allowed to remove distribution/architecture")
}

func (c *ClientBuilder) UpdateDistribution(d deb.Distribution, a deb.Architecture, output io.Writer) error {
	id, _, errChan, err := c.initSynchronization(output)
	if err != nil {
		return err
	}

	if err = c.conn.Call("RpcBuilder.Update", UpdateArgs{ID: id, Dist: d, Arch: a}, &NoValue{}); err != nil {
		return err
	}
	err = <-errChan
	return err
}

func (c *ClientBuilder) AvailableDistributions() []deb.Distribution {
	res := DistributionList{}
	if err := c.conn.Call("RpcBuilder.AvailableDistributions", NoValue{}, &res); err != nil {
		panic(err)
	}
	return res.Dists
}

func (c *ClientBuilder) AvailableArchitectures(d deb.Distribution) ArchitectureList {
	res := ArchitectureListReturn{}
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
	logger                             *log.Logger
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
	id := <-b.generator
	tmpFile := path.Join(os.TempDir(), fmt.Sprintf("go-deb.builder-%d-%d.sock", os.Getpid(), id))
	l, err := listenUnix(tmpFile)
	if err != nil {
		return err
	}

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

	b.logger.Printf("[%d]: Building package %s for distribution %s and architectures %s\n", args.ID,
		args.Args.SourcePackage.Identifier,
		args.Args.Dist,
		args.Args.Archs)
	actualRes, err := b.actualBuilder.BuildPackage(args.Args, s.w)
	b.logger.Printf("[%d]: Built %s, success:%v\n", args.ID, args.Args.SourcePackage.Identifier, err == nil)
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

	b.logger.Printf("[%d]: Creating distribution %s-%s\n", args.ID, args.Dist, args.Arch)
	err := b.actualBuilder.InitDistribution(args.Dist, args.Arch, writer)
	b.logger.Printf("[%d]: Created distribution %s-%s, success:%v\n", args.ID, args.Dist, args.Arch, err == nil)

	return err
}

type UpdateArgs struct {
	ID   SyncOutputID
	Dist deb.Distribution
	Arch deb.Architecture
}

func (b *RpcBuilder) Update(args UpdateArgs, res *NoValue) error {
	b.timeoutClearer <- args.ID
	s, ok := b.syncOutputs[args.ID]
	if ok == false {
		return fmt.Errorf("No output sunchronization %d available", args.ID)
	}
	defer func() { b.remover <- args.ID }()
	writer := s.w
	if writer == nil {
		return fmt.Errorf("Client is not connected to synchronization output %d", args.ID)
	}

	b.logger.Printf("[%d]: Updating distribution %s-%s", args.ID, args.Dist, args.Arch)
	err := b.actualBuilder.UpdateDistribution(args.Dist, args.Arch, writer)
	b.logger.Printf("[%d]: Updated distribution %s-%s, success:%v", args.ID, args.Dist, args.Arch, err == nil)
	return err
}

type DistributionList struct {
	Dists []deb.Distribution
}

func (b *RpcBuilder) AvailableDistributions(arg NoValue, res *DistributionList) error {
	res.Dists = b.actualBuilder.AvailableDistributions()
	return nil
}

type ArchitectureListReturn struct {
	Archs ArchitectureList
}

func (b *RpcBuilder) AvailableArchitectures(d deb.Distribution, res *ArchitectureListReturn) error {
	res.Archs = b.actualBuilder.AvailableArchitectures(d)
	return nil
}

type RpcBuilderServer struct {
	b       *RpcBuilder
	address string
	errChan chan error
	logger  *log.Logger
}

func NewRpcBuilderServer(builder DebianBuilder, address string) *RpcBuilderServer {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	return &RpcBuilderServer{
		b: &RpcBuilder{
			actualBuilder: builder,
			logger:        logger,
		},
		address: address,
		errChan: make(chan error),
		logger:  logger,
	}
}

func listenUnix(address string) (net.Listener, error) {

	l, err := net.Listen("unix", address)
	if err != nil {
		return nil, err
	}

	//TODO: use a special group for this
	fi, err := os.Stat(address)
	if err != nil {
		return nil, err
	}

	err = os.Chmod(address, fi.Mode()|0777)
	if err != nil {
		return nil, err
	}
	return l, nil
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

	l, err := listenUnix(s.address)
	if err != nil {
		s.errChan <- err
		return
	}

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt)
	go func() {
		for _ = range signals {
			s.Stop(l)
		}
	}()

	s.b.generator = make(chan SyncOutputID)
	s.b.timeouter = make(chan SyncOutputID)
	s.b.timeoutClearer = make(chan SyncOutputID)
	s.b.remover = make(chan SyncOutputID)
	s.b.syncOutputs = make(map[SyncOutputID]syncOutput)

	go s.b.manageSyncID()
	s.errChan <- nil
	s.logger.Printf("Started RPC builder on unix:/%s\n", s.address)
	http.Serve(l, nil)
}

func (s *RpcBuilderServer) Stop(l net.Listener) {
	l.Close()
	s.logger.Printf("Stopping RPC\n")
	s.logger.Printf("Removing unix:/%s\n", s.address)
	os.Remove(s.address)
}

func (s *RpcBuilderServer) WaitEstablished() error {
	return <-s.errChan
}
