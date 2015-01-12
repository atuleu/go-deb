package main

import (
	"fmt"
	"sort"

	deb ".."
	. "gopkg.in/check.v1"
)

type UserDistSupportConfigStub struct {
	supported map[deb.Distribution]map[deb.Architecture]bool
}

func (c *UserDistSupportConfigStub) Add(d deb.Distribution, a deb.Architecture) error {
	_, ok := c.supported[d]
	if ok == false {
		c.supported[d] = make(map[deb.Architecture]bool)
	}
	c.supported[d][a] = true
	return nil
}

func (c *UserDistSupportConfigStub) Remove(d deb.Distribution, a deb.Architecture) error {
	delete(c.supported[d], a)
	if len(c.supported[d]) == 0 {
		delete(c.supported, d)
	}
	return nil
}

func (c *UserDistSupportConfigStub) Supported() map[deb.Distribution]ArchitectureList {
	res := make(map[deb.Distribution]ArchitectureList)
	for d, archs := range c.supported {
		list := make(ArchitectureList, 0, len(archs))
		for a, _ := range archs {
			list = append(list, a)
		}
		sort.Sort(list)
		res[d] = list
	}
	return res
}

type DistManagementUseCaseSuite struct {
	x          *Interactor
	builder    *DebianBuilderStub
	distConfig *UserDistSupportConfigStub
	repo       *AptRepositoryStub
}

var _ = Suite(&DistManagementUseCaseSuite{})

func (s *DistManagementUseCaseSuite) SetUpSuite(c *C) {
	s.builder = &DebianBuilderStub{
		DistAndArch: make(map[deb.Distribution][]deb.Architecture),
	}
	s.distConfig = &UserDistSupportConfigStub{
		supported: make(map[deb.Distribution]map[deb.Architecture]bool),
	}
	s.repo = &AptRepositoryStub{}

	s.x = &Interactor{
		builder:         s.builder,
		userDistConfig:  s.distConfig,
		localRepository: s.repo,
	}

}

func (s *DistManagementUseCaseSuite) TestAddAndRemoveDistribution(c *C) {
	message, err := s.x.AddDistributionSupport("unstable", deb.Amd64, nil)
	c.Check(err, IsNil)
	c.Check(message.Message, Equals, "Builder initialized unstable-amd64\nEnabled user distribution support for unstable-amd64")
	message, err = s.x.AddDistributionSupport("unstable", deb.I386, nil)
	c.Check(err, IsNil)
	c.Check(message.Message, Equals, "Builder initialized unstable-i386\nEnabled user distribution support for unstable-i386")

	message, err = s.x.AddDistributionSupport("unstable", deb.Amd64, nil)
	c.Check(err, IsNil)
	c.Check(message.Message, Equals, "Enabled user distribution support for unstable-amd64")
	s.builder.Err = fmt.Errorf("I cannot cross-compile")

	message, err = s.x.AddDistributionSupport("unstable", deb.Armel, nil)
	c.Check(err, ErrorMatches, "I cannot cross-compile")
	c.Check(message.Message, Equals, "Builder could not initialize distribution unstable-armel")
	s.builder.Err = nil

	data, err := s.x.GetSupportedDistribution()
	c.Assert(err, IsNil)
	c.Assert(len(data), Equals, 1)
	archs := data[deb.Distribution("unstable")]
	c.Assert(len(archs), Equals, 2)
	c.Check(archs[deb.Amd64], Equals, true)
	c.Check(archs[deb.I386], Equals, true)

	err = s.x.RemoveDistributionSupport("buzz", deb.Amd64, false)
	c.Check(err, IsNil)
	err = s.x.RemoveDistributionSupport("unstable", deb.Amd64, false)
	c.Check(err, IsNil)

	err = s.x.RemoveDistributionSupport("unstable", deb.I386, true)
	c.Check(err, IsNil)

	data, err = s.x.GetSupportedDistribution()
	c.Assert(err, IsNil)
	c.Assert(len(data), Equals, 1)
	archs = data[deb.Distribution("unstable")]
	c.Check(len(archs), Equals, 1)
	v, ok := archs[deb.Amd64]
	c.Check(v, Equals, false)
	c.Check(ok, Equals, true)

	err = s.distConfig.Add("buzz", deb.Amd64)
	c.Assert(err, IsNil)

	data, err = s.x.GetSupportedDistribution()
	c.Check(data, IsNil)
	c.Check(err, ErrorMatches, "System consistency error: user list distributions buzz:.amd64., but builder does not support buzz")

	err = s.distConfig.Remove("buzz", deb.Amd64)
	c.Assert(err, IsNil)

	err = s.distConfig.Add("unstable", deb.I386)
	c.Assert(err, IsNil)
	err = s.distConfig.Add("unstable", deb.Amd64)
	c.Assert(err, IsNil)

	data, err = s.x.GetSupportedDistribution()
	c.Check(data, IsNil)
	c.Check(err, ErrorMatches, "System consistency error: user list distributions unstable:.amd64 i386., but builder does not support i386 for unstable")

}
