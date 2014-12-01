package deb

import . "gopkg.in/check.v1"

type ChangeFileSuite struct{}

var _ = Suite(&ChangeFileSuite{})

func (s *ChangeFileSuite) TestChangeFileRefName(c *C) {
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

func (s *ChangeFileSuite) TestBinaryPackageListing(c *C) {
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
