package main

import (
	"io/ioutil"
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
}

var _ = Suite(&RpcBuilderSuite{})

func (s *RpcBuilderSuite) SetUpSuite(c *C) {
	var err error
	s.tmpDir, err = ioutil.TempDir("", "go-deb.builder_test")
	c.Assert(err, IsNil)
	s.sock = path.Join(s.tmpDir, "rpc.sock")
	s.b = &DebianBuilderStub{
		DistAndArch: map[deb.Distribution][]deb.Architecture{
			deb.Distribution("unstable"): []deb.Architecture{deb.Amd64},
		},
	}
	s.s = NewRpcBuilderServer(s.b, s.sock)
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

	b, err := s.c.BuildPackage(args, nil)
	c.Check(err, IsNil)
	c.Check(b, NotNil)

}

func (s *RpcBuilderSuite) TestCreateAndRemove(c *C) {
	err := s.c.InitDistribution("sid", deb.Amd64, nil)
	c.Check(err, IsNil)
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
	c.Assert(err, IsNil)
	dists = s.c.AvailableDistributions()
	c.Assert(len(dists), Equals, 1)
	c.Check(dists[0], Equals, deb.Distribution("unstable"))
}

func (s *RpcBuilderSuite) TestUpdateDistribution(c *C) {
	err := s.c.UpdateDistribution("unstable", deb.Amd64)
	c.Check(err, IsNil)
	err = s.c.UpdateDistribution("sid", deb.Amd64)
	c.Check(err, ErrorMatches, "Distribution sid is not supported")
}
