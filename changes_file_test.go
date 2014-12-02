package deb

import (
	"bytes"
	"fmt"
	"net/mail"
	"strings"
	"time"

	. "gopkg.in/check.v1"
)

type ChangesFileSuite struct{}

var _ = Suite(&ChangesFileSuite{})

func (s *ChangesFileSuite) TestChangeFileRefName(c *C) {
	ch := ChangesFileRef{
		Identifier: SourcePackageRef{
			Source: "foo-bar",
			Ver: Version{
				Epoch:           3,
				UpstreamVersion: "1.2.3~4",
				DebianRevision:  "0ubuntu1",
			},
		},
		Suffix: "multi",
	}

	c.Check(ch.Filename(), Equals, "foo-bar_3:1.2.3~4-0ubuntu1_multi.changes")
}

func (s *ChangesFileSuite) TestBinaryPackageListing(c *C) {
	ch := ChangesFile{
		Md5Files: []FileReference{
			FileReference{Name: "libfoo_3:1.2.3~4-1.dsc"},
			FileReference{Name: "libfoo_1.2.3~4.orig.tar.gz"},
			FileReference{Name: "libfoo_3:1.2.3~4-1.debian.tar.gz"},
			FileReference{Name: "libfoo0_3:1.2.3~4-1_amd64.deb"},
			FileReference{Name: "libfoo-dev_3:1.2.3~4-1_amd64.deb"},
			FileReference{Name: "libfoo-dbg_3:1.2.3~4-1_amd64.deb"},
			FileReference{Name: "libfoo0_3:1.2.3~4-1_i386.deb"},
			FileReference{Name: "libfoo-dev_3:1.2.3~4-1_i386.deb"},
			FileReference{Name: "libfoo-dbg_3:1.2.3~4-1_i386.deb"},
			FileReference{Name: "libfoo-doc_3:1.2.3~4-1_all.deb"},
		},
	}

	ver := Version{
		Epoch:           3,
		UpstreamVersion: "1.2.3~4",
		DebianRevision:  "1",
	}

	bPackages, err := ch.BinaryPackages()
	if c.Check(err, IsNil, Commentf("Got unexpected error: %s", err)) == true {
		c.Assert(len(bPackages), Equals, 7)
		c.Check(bPackages[0], DeepEquals, BinaryPackageRef{Name: "libfoo0", Ver: ver, Arch: Amd64})
		c.Check(bPackages[1], DeepEquals, BinaryPackageRef{Name: "libfoo-dev", Ver: ver, Arch: Amd64})
		c.Check(bPackages[2], DeepEquals, BinaryPackageRef{Name: "libfoo-dbg", Ver: ver, Arch: Amd64})
		c.Check(bPackages[3], DeepEquals, BinaryPackageRef{Name: "libfoo0", Ver: ver, Arch: I386})
		c.Check(bPackages[4], DeepEquals, BinaryPackageRef{Name: "libfoo-dev", Ver: ver, Arch: I386})
		c.Check(bPackages[5], DeepEquals, BinaryPackageRef{Name: "libfoo-dbg", Ver: ver, Arch: I386})
		c.Check(bPackages[6], DeepEquals, BinaryPackageRef{Name: "libfoo-doc", Ver: ver, Arch: All})
	}

	invalidData := map[string]ChangesFile{
		"Invalid upstream version .*": ChangesFile{
			Md5Files: []FileReference{
				FileReference{Name: "libfoo_1.2.3:3-4-4_amd64.deb"},
			},
		},
		".changes file list invalid file `.*'": ChangesFile{
			Md5Files: []FileReference{
				FileReference{Name: "libfoo_1.2.3-1.deb"},
			},
		},
	}

	for errRegexp, ch := range invalidData {
		bPackage, err := ch.BinaryPackages()
		c.Check(bPackage, IsNil)
		c.Check(err, ErrorMatches, errRegexp)
	}
}

func (s *ChangesFileSuite) TestChangesFileParsing(c *C) {
	//This is the real content of a package generated by dpkg-genchanges
	changesFileContent := `Format: 1.8
Date: Tue, 10 Jun 2014 21:44:59 +0200
Source: aha
Binary: aha
Architecture: source amd64
Version: 0.4.7.2-1
Distribution: unstable
Urgency: medium
Maintainer: Axel Beckert <abe@debian.org>
Changed-By: Axel Beckert <abe@debian.org>
Description: 
 aha        - ANSI color to HTML converter
Changes: 
 aha (0.4.7.2-1) unstable; urgency=medium
 .
   * New upstream release
     + Drop sole patch. Merged upstream.
Checksums-Sha1: 
 8b2bac5c8d136e6532dd1183745a0e139f15ccf1 1806 aha_0.4.7.2-1.dsc
 09933fddb02b3129a690eb3d7d140edb97ac0627 6601 aha_0.4.7.2.orig.tar.gz
 34378e6c568a1716f716675664bc9e93dca96840 10969 aha_0.4.7.2-1.debian.tar.gz
 382dba0313117a92754e1a6557e5d7988479353b 20402 aha_0.4.7.2-1_amd64.deb
Checksums-Sha256: 
 cc020b7a4102dbd6101b11f208059e566d272234fda72eb03b77af2bd3dae6f8 1806 aha_0.4.7.2-1.dsc
 d07f92f4072b3222fe7577168f99a623ddeb4e88783dd132df41c5ffbdde16cf 6601 aha_0.4.7.2.orig.tar.gz
 9534d6e384345bbfffef5f367d8b6c27061fe9c0687262db4eb1c392b3a31e96 10969 aha_0.4.7.2-1.debian.tar.gz
 2d92d60188c6cd18298b5e8248b9728bb88e11bf3a0d17d322bf9265d1bd00da 20402 aha_0.4.7.2-1_amd64.deb
Files: 
 97ae4c309d12083da26e63f21a8189a8 1806 utils extra aha_0.4.7.2-1.dsc
 daeb9fc99362098340197c957645a877 6601 utils extra aha_0.4.7.2.orig.tar.gz
 7a3334229d270575947882b36856cee7 10969 utils extra aha_0.4.7.2-1.debian.tar.gz
 9240c714a75eb540871330f0fc454487 20402 utils extra aha_0.4.7.2-1_amd64.deb
`
	ch, err := ParseChangeFile(strings.NewReader(changesFileContent))
	c.Assert(err, IsNil)
	c.Assert(ch, NotNil)
	c.Check(ch.Format, DeepEquals, Version{0, "1.8", "0"})
	c.Check(ch.Format, DeepEquals, Version{0, "1.8", "0"})
	// we express all in UTC time
	location, err := time.LoadLocation("UTC")
	c.Assert(err, IsNil)
	c.Check(ch.Date, DeepEquals, time.Date(2014, time.June, 10, 19, 44, 59, 0, location))
	c.Check(ch.Ref.Identifier.Source, Equals, "aha")
	c.Check(ch.Binary, DeepEquals, []string{"aha"})
	c.Check(ch.Arch, DeepEquals, []Architecture{Source, Amd64})
	c.Check(ch.Ref.Identifier.Ver, DeepEquals, Version{0, "0.4.7.2", "1"})
	c.Check(ch.Dist, Equals, Distribution("unstable"))
	c.Check(ch.Maintainer, DeepEquals, &mail.Address{Name: "Axel Beckert", Address: "abe@debian.org"})
	c.Check(ch.Description, Equals, `aha        - ANSI color to HTML converter`)
	c.Check(ch.Changes, Equals, "aha (0.4.7.2-1) unstable; urgency=medium\n.\n* New upstream release\n+ Drop sole patch. Merged upstream.")
	c.Assert(len(ch.Sha1Files), Equals, 4)
	c.Assert(len(ch.Sha256Files), Equals, 4)
	c.Assert(len(ch.Md5Files), Equals, 4)
	for i := 0; i < 4; i = i + 1 {
		c.Check(ch.Md5Files[i].Size, Equals, ch.Sha1Files[i].Size)
		c.Check(ch.Md5Files[i].Size, Equals, ch.Sha256Files[i].Size)

		c.Check(ch.Md5Files[i].Name, Equals, ch.Sha1Files[i].Name)
		c.Check(ch.Md5Files[i].Name, Equals, ch.Sha256Files[i].Name)
	}
}

func (s *ChangesFileSuite) TestSingleLineFieldParse(c *C) {
	data := []string{"Format", "Version", "Date", "Distribution", "Source", "Version", "Maintainer"}

	for _, field := range data {
		ch, err := ParseChangeFile(strings.NewReader(fmt.Sprintf("%s: \n multi\n", field)))
		c.Check(ch, IsNil)
		c.Check(err, ErrorMatches, fmt.Sprintf("Invalid .changes field %s: .*: expected a single line field", field))
	}

}

func (s *ChangesFileSuite) TestMultiLineFieldParse(c *C) {
	data := []string{"Files", "Checksums-Sha1", "Checksums-Sha256", "Description", "Changes"}
	for _, field := range data {
		formats := []string{"%s: foo\n", "%s: foo\n bar\n baz\n"}
		for _, format := range formats {
			content := fmt.Sprintf(format, field)
			ch, err := ParseChangeFile(strings.NewReader(content))
			errMatch := fmt.Sprintf("Invalid .changes field %s: .*: expected a multi-line field, first line empty", field)
			c.Check(ch, IsNil)
			c.Check(err, ErrorMatches, errMatch)
		}
	}
}

func (s *ChangesFileSuite) TestParseChangesErrors(c *C) {
	data := map[string]string{
		".changes are expected to have a single paragraph": `Format: 1.8

Version: 1.2.3-1
`,
		"Unexpected .changes field Is-not-a-debian-field": `Is-not-a-debian-field: my-value
`,
		`Invalid .changes field Format: \[.*\]: it should have no epoch or debian revision`: `Format: 3:1.3-1
`,
		"Invalid .changes field Format: .*: Invalid upstream version `1.3:3', it should not contain a colon since epoch is 0.*": `Format: 1.3:3-1
`,
		"Invalid .changes field Version: .*: Invalid upstream version `1.3:3', it should not contain a colon since epoch is 0.*": `Version: 1.3:3-1
`,
		"Invalid .changes field Files: .*: invalid line `.*' .4 elements., expected `checksum size .section priority. name'": `Files:
 0123456789abcdef 123 extra-section file.deb
`,
		"Invalid .changes field Files: .*: encoding/hex.*": `Files:
 01234567x9abcdef 123 file.deb
`,
		"Invalid .changes field Files: .*: expected integer": `Files:
 012345679abcdef0 e123 file.deb
`,
		"Invalid .changes field Architecture: .*: unknown architecture notanarch": `Architecture: source notanarch
`,
		"Invalid .changes field Distribution: .*: does not contains a single distribution": `Distribution: source notanarch
`,
		"Invalid .changes field Maintainer: .*: mail: .*": `Maintainer: source
`,
		"Invalid .changes field Source: .*: multiple source name": `Source: foo bar
`,
		"Invalid .changes field Date: .*: parsing time .*": `Date: foo +0000
`,
		".changes parse error: .*": ` Urgency:`,
	}
	for errMatch, content := range data {
		ch, err := ParseChangeFile(strings.NewReader(content))
		c.Check(ch, IsNil)
		c.Check(err, ErrorMatches, errMatch)
	}
}

func (s *ChangesFileSuite) TestRequiredFieldParse(c *C) {
	data := map[string]string{
		"Format":           "1.8",
		"Date":             "Tue, 10 Jun 2014 21:44:59 -0200",
		"Source":           "aha",
		"Binary":           "aha",
		"Architecture":     "source amd64",
		"Version":          "0.4.7.2-1",
		"Distribution":     "unstable",
		"Urgency":          "medium",
		"Maintainer":       "Axel Beckert <abe@debian.org>",
		"Changed-By":       "Axel Beckert <abe@debian.org>",
		"Description":      "\n aha        - ANSI color to HTML converter",
		"Changes":          "\n aha (0.4.7.2-1) unstable; urgency=medium\n .\n   * New upstream release\n     + Drop sole patch. Merged upstream.",
		"Checksums-Sha1":   "\n  8b2bac5c8d136e6532dd1183745a0e139f15ccf1 1806 aha_0.4.7.2-1.dsc",
		"Checksums-Sha256": "\n cc020b7a4102dbd6101b11f208059e566d272234fda72eb03b77af2bd3dae6f8 1806 aha_0.4.7.2-1.dsc",
		"Files":            "\n 97ae4c309d12083da26e63f21a8189a8 1806 utils extra aha_0.4.7.2-1.dsc",
	}

	requiredField := []string{"Format", "Date", "Source", "Binary", "Version", "Architecture", "Distribution", "Maintainer", "Description", "Changes", "Checksums-Sha1", "Checksums-Sha256", "Files"}
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
		ch, err := ParseChangeFile(&content)
		c.Check(ch, IsNil)
		c.Check(err, ErrorMatches, ".changes miss mandatory field "+field)
	}

}

func (s *ChangesFileSuite) TestDateOffsetParsing(c *C) {
	invalid := []string{"+00000", "!0000", "+u000", "-00u0", "+5662"}

	for _, off := range invalid {
		ch, err := ParseChangeFile(strings.NewReader("Date: foo " + off))
		c.Check(ch, IsNil)
		c.Check(err, ErrorMatches, "Invalid .changes field Date: .*: invalid UTC offset `.*'")
	}
}
