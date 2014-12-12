package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"

	deb ".."
	"golang.org/x/crypto/openpgp/clearsign"
	. "gopkg.in/check.v1"
)

type XdgArchiverSuite struct {
	h        TempHomer
	archiver *XdgArchiver
	dscFile  *deb.SourceControlFile
}

var _ = Suite(&XdgArchiverSuite{})

func (s *XdgArchiverSuite) SetUpSuite(c *C) {
	s.h.OverrideEnv("XDG_DATA_HOME", "")
	err := s.h.SetUp()
	c.Assert(err, IsNil)

	auth, err := NewAuthentifier()
	c.Assert(err, IsNil)
	s.archiver, err = NewXdgArchiver(auth)
	c.Assert(err, IsNil)

	cmd := exec.Command("apt-get", "source", "aha")
	cmd.Stdin = nil

	home := os.Getenv("HOME")
	cmd.Dir = home
	out, err := cmd.CombinedOutput()
	c.Assert(err, IsNil)
	rx := regexp.MustCompile(`^Get:[0-9]+ .* .* aha (.*) \(dsc\) \[.*\]\n$`)
	r := bytes.NewBuffer(out)
	var ver *deb.Version = nil
	stop := false
	for stop == false {
		l, err := r.ReadString('\n')
		if err != io.EOF {
			c.Assert(err, IsNil)
		}
		if err == io.EOF {
			stop = true
		}
		m := rx.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		ver, err = deb.ParseVersion(m[1])
		c.Assert(err, IsNil)
		break
	}

	c.Assert(ver, NotNil)
	f, err := os.Open(path.Join(home, fmt.Sprintf("aha_%s.dsc", *ver)))
	c.Assert(err, IsNil)
	allData, err := ioutil.ReadAll(f)
	c.Assert(err, IsNil)
	block, rest := clearsign.Decode(allData)

	var unsigned io.Reader = nil
	if block == nil {
		unsigned = bytes.NewReader(rest)
	} else {
		unsigned = bytes.NewReader(block.Plaintext)
	}
	s.dscFile, err = deb.ParseDsc(unsigned)
	c.Assert(err, IsNil)
	s.dscFile.BasePath = home
}

func (s *XdgArchiverSuite) TearDown(c *C) {
	err := s.h.TearDown()
	c.Assert(err, IsNil)
}

func (s *XdgArchiverSuite) TestArchiveSource(c *C) {
	_, err := s.archiver.ArchiveSource(*s.dscFile)
	c.Check(err, IsNil)
}
