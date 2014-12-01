package main

import (
	"io/ioutil"
	"os"

	deb ".."
	. "gopkg.in/check.v1"
)

type TempHomer struct {
	envOverrides map[string]string
	envSaves     map[string]string
	tmpDir       string
}

func (h *TempHomer) SetUp() error {
	var err error
	h.tmpDir, err = ioutil.TempDir("", "go-deb_builder_test")
	if err != nil {
	}
	h.OverrideEnv("HOME", h.tmpDir)

	h.envSaves = make(map[string]string)
	for key, value := range h.envOverrides {
		h.envSaves[key] = os.Getenv(key)
		os.Setenv(key, value)
	}
	return nil
}

func (h *TempHomer) TearDown() error {
	for key, value := range h.envSaves {
		os.Setenv(key, value)
	}
	return os.RemoveAll(h.tmpDir)
}

func (s *TempHomer) OverrideEnv(key, value string) {
	if s.envOverrides == nil {
		s.envOverrides = make(map[string]string)
	}
	s.envOverrides[key] = value
}

type LocalRepreproSuite struct {
	h TempHomer
	r *LocalReprepro
}

var _ = Suite(&LocalRepreproSuite{})

func (s *LocalRepreproSuite) SetUpSuite(c *C) {
	s.h.OverrideEnv("XDG_DATA_HOME", "")
	err := s.h.SetUp()
	c.Assert(err, IsNil, Commentf("Initialization error %s", err))

	s.r, err = NewLocalReprepro()
	c.Assert(err, IsNil, Commentf("Initialization error %s", err))

	err = s.r.AddDistribution(DistributionAndArch{Dist: "unstable", Arch: deb.Amd64})
	c.Assert(err, IsNil, Commentf("Initialization error %s", err))
}

func (s *LocalRepreproSuite) TearDownSuite(c *C) {
	err := s.h.TearDown()
	c.Assert(err, IsNil, Commentf("Cleanup error %s", err))
}

func (s *LocalRepreproSuite) TestLockFailure(c *C) {
	//acquire the lock
	err := s.r.tryLock()
	c.Assert(err, IsNil, Commentf("Unexpected error: %s", err))

	c.Check(s.r.ListPackage("foo", nil), IsNil)
	errMatch := "Could not lock repository .*: Locked by other process"
	c.Check(s.r.AddDistribution(DistributionAndArch{Dist: "foo", Arch: deb.Amd64}), ErrorMatches, errMatch)
	c.Check(s.r.RemoveDistribution(DistributionAndArch{Dist: "foo", Arch: deb.Amd64}), ErrorMatches, errMatch)
	b := &BuildResult{
		Changes: &deb.ChangesFile{
			Dist: "unstable",
		},
	}

	c.Check(s.r.ArchiveBuildResult(b), ErrorMatches, errMatch)
	c.Check(s.r.RemovePackage("unstable", deb.BinaryPackageRef{}), ErrorMatches, errMatch)
}
