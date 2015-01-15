package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path"

	deb ".."
	. "gopkg.in/check.v1"
)

type RpcBuilderSuite struct {
	b            *DebianBuilderStub
	s            *RpcBuilderServer
	c            *ClientBuilder
	tmpDir, sock string
	output       bytes.Buffer
}

var _ = Suite(&RpcBuilderSuite{})

func (s *RpcBuilderSuite) SetUpSuite(c *C) {
	var err error
	s.tmpDir, err = ioutil.TempDir("", "go-deb.builder_test")
	c.Assert(err, IsNil)
	s.sock = path.Join(s.tmpDir, "rpc.sock")
	s.b = &DebianBuilderStub{
		DistAndArch: map[deb.Codename][]deb.Architecture{
			deb.Codename("unstable"): []deb.Architecture{deb.Amd64},
		},
	}

	s.s = NewRpcBuilderServer(s.b, s.sock)
	//we remove output from tests
	//TODO: we coudl unit test the logging now
	voidLogger := log.New(&s.output, s.s.logger.Prefix(), s.s.logger.Flags())
	s.s.b.logger = voidLogger
	s.s.logger = voidLogger

	go s.s.Serve()
	err = s.s.WaitEstablished()
	c.Assert(err, IsNil)
	s.c, err = NewClientBuilder("unix", s.sock)
	c.Assert(err, IsNil)
}

func (s *RpcBuilderSuite) TearDownSuite(c *C) {
	err := os.RemoveAll(s.tmpDir)
	c.Assert(err, IsNil)
}

func (s *RpcBuilderSuite) TestConnection(c *C) {}

func (s *RpcBuilderSuite) TestBuild(c *C) {
	dsc := deb.SourceControlFile{
		Identifier: deb.SourcePackageRef{
			Source: "foo",
			Ver: deb.Version{
				UpstreamVersion: "1.2.3",
				DebianRevision:  "1",
			},
		},
	}

	args := BuildArguments{
		SourcePackage: dsc,
		Dist:          "unstable",
		Archs:         []deb.Architecture{deb.Amd64},
		Deps:          nil,
	}
	var out bytes.Buffer
	b, err := s.c.BuildPackage(args, &out)
	c.Check(err, IsNil)
	c.Check(b, NotNil)
	c.Check(out.String(), Equals, "Called BuildPackage\n")

}

func (s *RpcBuilderSuite) TestCreateAndRemove(c *C) {
	var out bytes.Buffer
	err := s.c.InitDistribution("sid", deb.Amd64, &out)
	c.Check(err, IsNil)
	c.Check(out.String(), Equals, "Called InitDistribution\n")

	dists := s.c.AvailableDistributions()
	c.Check(len(dists), Equals, 2)
	for _, d := range dists {
		archs := s.c.AvailableArchitectures(d)
		if c.Check(len(archs), Equals, 1) == false {
			continue
		}
		c.Check(archs[0], Equals, deb.Amd64)
	}

	err = s.c.RemoveDistribution("sid", deb.Amd64)
	c.Assert(err, ErrorMatches, "Client builder are not allowed to remove distribution/architecture")
	dists = s.c.AvailableDistributions()
}

func (s *RpcBuilderSuite) TestUpdateDistribution(c *C) {
	var out bytes.Buffer
	err := s.c.UpdateDistribution("unstable", deb.Amd64, &out)
	c.Check(err, IsNil)
	c.Check(out.String(), Equals, "Called UpdateDistribution\n")
	//cannot use sid in that example, it may or may not have been
	//added by other test
	err = s.c.UpdateDistribution("buzz", deb.Amd64, nil)
	c.Check(err, ErrorMatches, "Distribution buzz is not supported")
}
