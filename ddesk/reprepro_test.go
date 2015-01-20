package main

import (
	"io/ioutil"
	"os"
	"path"

	"launchpad.net/go-xdg"

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
	h.tmpDir, err = ioutil.TempDir("", "go-deb.ddesk_test")
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

type RepreproSuite struct {
	h        TempHomer
	r        *Reprepro
	repoPath string
}

var _ = Suite(&RepreproSuite{})

func (s *RepreproSuite) SetUpSuite(c *C) {
	s.h.OverrideEnv("XDG_DATA_HOME", "")
	err := s.h.SetUp()
	c.Assert(err, IsNil, Commentf("Initialization error %s", err))

	s.repoPath = path.Join(xdg.Data.Home(), "go-deb.ddesk/reprepro_test")
	s.r, err = NewReprepro(s.repoPath)
	c.Assert(err, IsNil, Commentf("Initialization error %s", err))

	err = s.r.AddDistribution(deb.Unstable, deb.Amd64)
	c.Assert(err, IsNil, Commentf("Initialization error %s", err))
	c.Assert(len(s.r.dists[deb.Unstable]), Equals, 1)
	c.Assert(s.r.dists[deb.Unstable][deb.Amd64], Equals, true)
}

func (s *RepreproSuite) TearDownSuite(c *C) {
	err := s.h.TearDown()
	c.Assert(err, IsNil, Commentf("Cleanup error %s", err))
}

func (s *RepreproSuite) TestLockFailure(c *C) {
	//acquire the lock
	err := s.r.tryLock()
	c.Assert(err, IsNil, Commentf("Unexpected error: %s", err))

	c.Check(s.r.ListPackage(deb.Unstable, nil), IsNil)
	errMatch := "Could not lock repository .*: Locked by other process"
	c.Check(s.r.AddDistribution("foo", deb.Amd64), ErrorMatches, errMatch)
	c.Check(s.r.RemoveDistribution(deb.Unstable, deb.Amd64), ErrorMatches, errMatch)
	c.Assert(s.r.dists[deb.Unstable], NotNil)
	c.Assert(s.r.dists[deb.Unstable][deb.Amd64], Equals, true)
	b := &BuildResult{
		Changes: &deb.ChangesFile{
			Dist: "unstable",
		},
	}

	c.Check(s.r.ArchiveChanges(b.Changes, os.TempDir()), ErrorMatches, errMatch)
	c.Check(s.r.RemovePackage("unstable", deb.BinaryPackageRef{}), ErrorMatches, errMatch)
	defer func() {
		c.Assert(recover(), IsNil)
	}()
	s.r.unlockOrPanic()
}

func (s *RepreproSuite) TestReload(c *C) {
	newRepo, err := NewReprepro(s.repoPath)
	c.Check(err, IsNil)
	c.Check(newRepo, DeepEquals, s.r)
}

func (s *RepreproSuite) TestCanAddSameArchitectureTwice(c *C) {
	err := s.r.AddDistribution(deb.Unstable, deb.Amd64)
	c.Assert(err, IsNil, Commentf("Initialization error %s", err))

}
