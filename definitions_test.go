package deb

import . "gopkg.in/check.v1"

type DefinitionsSuite struct {
}

var _ = Suite(&DefinitionsSuite{})

func (s *DefinitionsSuite) TestSourcePackageRefName(c *C) {
	pr := SourcePackageRef{
		Source: "foo",
		Ver: Version{
			UpstreamVersion: "1.2.3",
			DebianRevision:  "1",
		},
	}

	c.Check(pr.String(), Equals, "foo_1.2.3-1")

	validData := map[string]*SourcePackageRef{
		"/b/c/foo_2:1.2.3-1.dsc": &SourcePackageRef{
			Source: "foo",
			Ver:    Version{Epoch: 2, UpstreamVersion: "1.2.3", DebianRevision: "1"},
		},
		"foo_2:1.2.3-1.debian.tar.gz": &SourcePackageRef{
			Source: "foo",
			Ver:    Version{Epoch: 2, UpstreamVersion: "1.2.3", DebianRevision: "1"},
		},
	}

	for path, expected := range validData {
		res, err := NewRefFromFileName(path)
		c.Check(err, IsNil)
		if c.Check(res, NotNil) == false {
			continue
		}
		c.Check(res, DeepEquals, expected)
	}

	invalidData := map[string]string{
		"foo_1.2.3-1.gif": "Invalid file name .*",
		"foo_1.2.3:3.dsc": "Invalid upstream version.*",
	}

	for path, errMatch := range invalidData {
		res, err := NewRefFromFileName(path)
		c.Check(res, IsNil)
		c.Check(err, ErrorMatches, errMatch)
	}
}
