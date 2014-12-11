package deb

import (
	"bytes"
	"fmt"
	"net/mail"
	"strings"

	. "gopkg.in/check.v1"
)

type SourceControlFileSuite struct {
}

var _ = Suite(&SourceControlFileSuite{})

func (s *SourceControlFileSuite) TestSourceControlFileName(c *C) {
	dsc := SourceControlFile{
		Identifier: SourcePackageRef{
			Source: "foo",
			Ver: Version{
				Epoch:           3,
				DebianRevision:  "2er3",
				UpstreamVersion: "1.2.3.4.5",
			},
		},
	}

	c.Check(dsc.Filename(), Equals, fmt.Sprintf("%s_%s.dsc", dsc.Identifier.Source, dsc.Identifier.Ver))
	c.Check(dsc.ChangesFilename(), Equals, fmt.Sprintf("%s_%s_source.changes", dsc.Identifier.Source, dsc.Identifier.Ver))

	c.Check(IsDscFileName(dsc.Filename()), IsNil)

	invalid := map[string]string{
		"/a/b/c/foo_1.2:3:3~foo-1.dsc": "Invalid upstream version `1.2:3:3~foo', it should not contain a colon since epoch is 0",
		"a/b/c/foo_1.23~foo-1.gif":     "Wrong filename syntax foo_1.23~foo-1.gif",
	}

	for path, errMatch := range invalid {
		c.Check(IsDscFileName(path), ErrorMatches, errMatch)
	}
}

func (s *SourceControlFileSuite) TestSourceControlFileParsing(c *C) {
	dscContent := `Format: 3.0 (quilt)
Source: aha
Binary: aha
Architecture: any
Version: 0.4.4-1
Maintainer: Axel Beckert <abe@debian.org>
Homepage: http://ziz.delphigl.com/tool_aha.php
Standards-Version: 3.9.2
Vcs-Browser: http://git.debian.org/?p=collab-maint/aha.git;a=summary
Vcs-Git: git://git.debian.org/collab-maint/aha.git
Build-Depends: debhelper (>= 7)
Checksums-Sha1: 
 d5b5a18faffaffef0af03a96536afe73ad294db1 5518 aha_0.4.4.orig.tar.gz
 9331becfdefa01f3a24d07b661d779079ace7657 2245 aha_0.4.4-1.debian.tar.gz
Checksums-Sha256: 
 fdaa68efcff2f93598522143891cc69b2aae2329a18f6cbd307450d2e66e53d6 5518 aha_0.4.4.orig.tar.gz
 5340a5a23313cd472396cf160bd909b4e013a4ccdabaeb7e5650411db5455d65 2245 aha_0.4.4-1.debian.tar.gz
Files: 
 d9eb4bb38090193c02b28f92149f1ea6 5518 aha_0.4.4.orig.tar.gz
 b81d60ca7f47d88632b4b5332b602c0e 2245 aha_0.4.4-1.debian.tar.gz
`

	dsc, err := ParseDsc(strings.NewReader(dscContent))
	c.Check(err, IsNil)
	c.Assert(dsc, NotNil)
	c.Check(dsc.Identifier.Source, Equals, "aha")
	c.Check(dsc.Identifier.Ver, DeepEquals, Version{UpstreamVersion: "0.4.4", DebianRevision: "1"})
	c.Check(dsc.Maintainer, DeepEquals, &mail.Address{Name: "Axel Beckert", Address: "abe@debian.org"})
}

func (s *SourceControlFileSuite) TestSourceControlFileParseError(c *C) {

	invalid := map[string]string{
		`Format: 3.0 foo`: "invalid field Format:.*: invalid format .*",
		`Format: 3.0 foo
 truc`: "invalid field Format:.*: expected a single line field",
	}

	for content, errMatch := range invalid {
		dsc, err := ParseDsc(strings.NewReader(content + "\n"))
		c.Check(dsc, IsNil)
		c.Check(err, ErrorMatches, ".dsc parse error: "+errMatch)
	}
}

func (s *SourceControlFileSuite) TestSourceControlFileParseRequiredField(c *C) {
	data := map[string]string{
		"Format":           "1.0",
		"Source":           "aha",
		"Vcs-Git":          "unused",
		"Version":          "0.4.7.2-1",
		"Maintainer":       "Axel Beckert <abe@debian.org>",
		"Checksums-Sha1":   "\n  8b2bac5c8d136e6532dd1183745a0e139f15ccf1 1806 aha_0.4.7.2-1.dsc",
		"Checksums-Sha256": "\n cc020b7a4102dbd6101b11f208059e566d272234fda72eb03b77af2bd3dae6f8 1806 aha_0.4.7.2-1.dsc",
		"Files":            "\n 97ae4c309d12083da26e63f21a8189a8 1806 utils extra aha_0.4.7.2-1.dsc",
	}

	requiredField := []string{"Format", "Source", "Version", "Maintainer", "Checksums-Sha1", "Checksums-Sha256", "Files"}
	for _, field := range requiredField {
		var content bytes.Buffer
		removed := false
		for f, d := range data {
			if f == field {
				removed = true
				continue
			}
			fmt.Fprintf(&content, "%s: %s\n", f, d)
		}
		if c.Check(removed, Equals, true, Commentf("Internal test error")) == false {
			continue
		}
		ch, err := ParseDsc(&content)
		c.Check(ch, IsNil)
		c.Check(err, ErrorMatches, ".dsc parse error: missing required field ."+field+".")
	}

}
