package main

import (
	deb ".."
	. "gopkg.in/check.v1"
)

type UserDistSupportConfigSuite struct {
	h    TempHomer
	xdg  *XdgUserDistConfig
	stub *UserDistSupportConfigStub
}

var _ = Suite(&UserDistSupportConfigSuite{})

func testDistConfig(u UserDistSupportConfig, c *C) {
	c.Assert(u, NotNil)

	err := u.Add("unstable", deb.Amd64)
	c.Check(err, IsNil)

	err = u.Add("unstable", deb.I386)
	c.Check(err, IsNil)

	err = u.Add("unstable", deb.Amd64)
	c.Check(err, IsNil)

	err = u.Add("unstable", deb.I386)
	c.Check(err, IsNil)

	supported := u.Supported()
	c.Assert(supported, NotNil)
	archs := supported[deb.Distribution("unstable")]

	c.Check(len(archs), Equals, 2)
	if len(archs) >= 2 {
		c.Check(archs[0], Equals, deb.Amd64)
		c.Check(archs[1], Equals, deb.I386)
	}
}

func (s *UserDistSupportConfigSuite) SetUpSuite(c *C) {
	s.h.OverrideEnv("XDG_DATA_HOME", "")
	err := s.h.SetUp()
	c.Assert(err, IsNil)

	s.xdg, err = NewXdgUserDistConfig()
	c.Assert(err, IsNil)

	s.stub = &UserDistSupportConfigStub{
		supported: make(map[deb.Distribution]map[deb.Architecture]bool),
	}

}

func (s *UserDistSupportConfigSuite) TearDownSuite(c *C) {
	err := s.h.TearDown()
	c.Assert(err, IsNil)
}

func (s *UserDistSupportConfigSuite) TestStub(c *C) {
	testDistConfig(s.stub, c)
}

func (s *UserDistSupportConfigSuite) TestXdg(c *C) {
	testDistConfig(s.xdg, c)
}
