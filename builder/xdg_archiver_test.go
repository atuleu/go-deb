package main

import (
	"io"
	"os"
	"path"
	"strings"

	deb ".."
	. "gopkg.in/check.v1"
)

type XdgArchiverSuite struct {
	h        TempHomer
	archiver *XdgArchiver
	dscFile  deb.SourceControlFile
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

	filesToWrite := map[string]*string{
		"aha_0.4.4-1.dsc":            &ahaDscContent,
		"aha_0.4.4-1_source.changes": &ahaChangesContent,
		"aha_0.4.4.orig.tar.gz":      nil,
		"aha_0.4.4-1.debian.tar.gz":  nil,
	}
	home := os.Getenv("HOME")
	for name, data := range filesToWrite {
		f, err := os.Create(path.Join(home, name))
		c.Assert(err, IsNil)
		defer f.Close()
		if data == nil {
			continue
		}
		_, err = io.Copy(f, strings.NewReader(*data))
		c.Assert(err, IsNil)
	}
	s.dscFile = deb.SourceControlFile{
		Identifier: deb.SourcePackageRef{
			Source: "aha",
			Ver: deb.Version{
				UpstreamVersion: "0.4.4",
				DebianRevision:  "1",
			},
		},
		BasePath: home,
	}
}

func (s *XdgArchiverSuite) TearDown(c *C) {
	err := s.h.TearDown()
	c.Assert(err, IsNil)
}

var ahaDscContent = `Format: 3.0 (quilt)
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

var ahaChangesContent = `Format: 1.8
Date: Wed, 31 Aug 2011 23:17:58 +0200
Source: aha
Binary: aha
Architecture: source
Version: 0.4.4-1
Distribution: unstable
Urgency: low
Maintainer: Axel Beckert <abe@debian.org>
Changed-By: Axel Beckert <abe@debian.org>
Description: 
 aha        - ANSI color to HTML converter
Changes: 
 aha (0.4.4-1) unstable; urgency=low
 .
   * New upstream release
     + Fixes issues with underlined and inverse text
   * Fix lintian warning helper-templates-in-copyright
   * Fix lintian warning debian-rules-missing-recommended-target
Checksums-Sha1: 
 3079c94e43135f8afa7f1fb55d42a855f09ffb91 1093 aha_0.4.4-1.dsc
 d5b5a18faffaffef0af03a96536afe73ad294db1 5518 aha_0.4.4.orig.tar.gz
 9331becfdefa01f3a24d07b661d779079ace7657 2245 aha_0.4.4-1.debian.tar.gz
Checksums-Sha256: 
 cb3fd434c94c0d53faf6520cbd7ffefb21cd7a90c99023d17b0a65e655d82bd0 1093 aha_0.4.4-1.dsc
 fdaa68efcff2f93598522143891cc69b2aae2329a18f6cbd307450d2e66e53d6 5518 aha_0.4.4.orig.tar.gz
 5340a5a23313cd472396cf160bd909b4e013a4ccdabaeb7e5650411db5455d65 2245 aha_0.4.4-1.debian.tar.gz
Files: 
 07a8ea4a93bd8ac11cb110b672811f79 1093 utils extra aha_0.4.4-1.dsc
 d9eb4bb38090193c02b28f92149f1ea6 5518 utils extra aha_0.4.4.orig.tar.gz
 b81d60ca7f47d88632b4b5332b602c0e 2245 utils extra aha_0.4.4-1.debian.tar.gz
`

func (s *XdgArchiverSuite) TestArchiveSource(c *C) {
	_, err := s.archiver.ArchiveSource(s.dscFile)
	c.Check(err, IsNil)
}
