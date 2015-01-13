package main

import (
	"io/ioutil"
	"os"
	"testing"

	. "gopkg.in/check.v1"
)

type UseCaseSuite struct {
	x Interactor
}

var _ = Suite(&UseCaseSuite{})

func Test(t *testing.T) { TestingT(t) }

func (s *UseCaseSuite) TestChangeFileProcessing(c *C) {
	res, err := s.x.ProcessChangesFile("")
	c.Check(res, NotNil)
	c.Check(res.SendTo, IsNil)
	c.Check(len(res.FilesToRemove), Equals, 0)
	c.Check(err, ErrorMatches, "Invalid filename .*")

	f, err := ioutil.TempFile("", "test")
	defer os.Remove(f.Name())

	res, err = s.x.ProcessChangesFile(f.Name() + ".changes")
	c.Check(res, NotNil)
	c.Check(res.SendTo, IsNil)
	c.Check(len(res.FilesToRemove), Equals, 0)
	c.Check(err, ErrorMatches, "open .*: no such file or directory")

}
