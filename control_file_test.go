package deb

import (
	"strings"

	. "gopkg.in/check.v1"
)

type ControlFileLexerSuite struct{}

var _ = Suite(&ControlFileLexerSuite{})

func (s *ControlFileLexerSuite) TestCanLexControlFile(c *C) {
	unsignedControlFile := `

Hash: SHA1

Format: 3.0 (quilt)
Maintainer: Alexandre Tuleu <alexandre.tuleu.2005@polytechnique.org>
Standards-Version: 3.9.3
Vcs-Browser: http://googlemock.googlecode.com/svn/trunk/
Build-Depends: debhelper (>= 8.0.0), cmake, 
 foo, bar
Checksums-Sha1: 
 c178c363b85f51caf01d2a2d2c86b48df417a60d 1786622 gmock_1.6.0.orig.tar.gz
 d030592231249a9b158eda57beaff197220f8fd2 2726 gmock_1.6.0-2.debian.tar.gz
`
	fields := []ControlField{
		ControlField{},
		ControlField{Name: "Hash", Data: []string{"SHA1"}},
		ControlField{},
		ControlField{Name: "Format", Data: []string{"3.0 (quilt)"}},
		ControlField{Name: "Maintainer", Data: []string{"Alexandre Tuleu <alexandre.tuleu.2005@polytechnique.org>"}},
		ControlField{Name: "Standards-Version", Data: []string{"3.9.3"}},
		ControlField{Name: "Vcs-Browser", Data: []string{"http://googlemock.googlecode.com/svn/trunk/"}},
		ControlField{Name: "Build-Depends", Data: []string{
			"debhelper (>= 8.0.0), cmake,",
			"foo, bar",
		}},
		ControlField{Name: "Checksums-Sha1", Data: []string{
			"",
			"c178c363b85f51caf01d2a2d2c86b48df417a60d 1786622 gmock_1.6.0.orig.tar.gz",
			"d030592231249a9b158eda57beaff197220f8fd2 2726 gmock_1.6.0-2.debian.tar.gz",
		}},
	}

	r := strings.NewReader(unsignedControlFile)
	l := NewControlFileLexer(r)
	for _, expected := range fields {
		f, err := l.Next()
		c.Assert(err, IsNil, Commentf("Got unexpected error %s", err))
		c.Check(f, DeepEquals, expected)

	}

}
