package main

import (
	deb ".."
	. "gopkg.in/check.v1"
)

type XdgHistorySuite struct {
	h    TempHomer
	hist *XdgHistory
}

var _ = Suite(&XdgHistorySuite{})

func (s *XdgHistorySuite) SetUpSuite(c *C) {
	s.h.OverrideEnv("XDG_DATA_HOME", "")
	err := s.h.SetUp()
	c.Assert(err, IsNil)

	s.hist, err = NewXdgHistory()
	c.Assert(err, IsNil)
}

func (s *XdgHistorySuite) TearDown(c *C) {
	err := s.h.TearDown()
	c.Assert(err, IsNil)
}

func (s *XdgHistorySuite) TestHistory(c *C) {
	a := deb.SourcePackageRef{
		Source: "a",
		Ver: deb.Version{
			Epoch:           0,
			UpstreamVersion: "1.2.3",
			DebianRevision:  "1",
		},
	}
	b := a
	aRc := a
	aRc.Ver.DebianRevision = "1~rc1"
	b.Source = "b"

	defer func() {
		r := recover()
		c.Assert(r, IsNil)
	}()

	s.hist.Append(aRc)
	s.hist.Append(a)
	s.hist.Append(b)
	s.hist.Append(a)
	s.hist.Append(a)

	c.Assert(len(s.hist.Get()), Equals, 5)
	c.Check(s.hist.Get()[0], DeepEquals, a)
	c.Check(s.hist.Get()[1], DeepEquals, a)
	c.Check(s.hist.Get()[2], DeepEquals, b)
	c.Check(s.hist.Get()[3], DeepEquals, a)
	c.Check(s.hist.Get()[4], DeepEquals, aRc)

	s.hist.RemoveFront(b)

	c.Assert(len(s.hist.Get()), Equals, 5)
	c.Check(s.hist.Get()[0], DeepEquals, a)
	c.Check(s.hist.Get()[1], DeepEquals, a)
	c.Check(s.hist.Get()[2], DeepEquals, b)
	c.Check(s.hist.Get()[3], DeepEquals, a)
	c.Check(s.hist.Get()[4], DeepEquals, aRc)

	s.hist.RemoveFront(a)

	c.Assert(len(s.hist.Get()), Equals, 3)
	c.Check(s.hist.Get()[0], DeepEquals, b)
	c.Check(s.hist.Get()[1], DeepEquals, a)
	c.Check(s.hist.Get()[2], DeepEquals, aRc)

}
