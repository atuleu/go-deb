package deb

import (
	"testing"

	. "gopkg.in/check.v1"
)

type VersionSuite struct{}

var _ = Suite(&VersionSuite{})

func Test(t *testing.T) { TestingT(t) }

func (s *VersionSuite) TestVersionParsing(c *C) {
	validData := map[string]*Version{
		"1.2.3~4-1": &Version{
			Epoch:           0,
			UpstreamVersion: "1.2.3~4",
			DebianRevision:  "1",
		},
		"3:1.2.3~4-1": &Version{
			Epoch:           3,
			UpstreamVersion: "1.2.3~4",
			DebianRevision:  "1",
		},
		"1.fooBar.3~4": &Version{
			Epoch:           0,
			UpstreamVersion: "1.fooBar.3~4",
			DebianRevision:  "0",
		},
	}

	for s, expected := range validData {
		v, err := ParseVersion(s)
		if c.Check(err, IsNil, Commentf("Got unexpected error: %s", err)) == false {
			continue
		}
		c.Check(v, DeepEquals, expected)
	}
}

func (s *VersionSuite) TestVersionWithNoEpochContainsNoColons(c *C) {
	v, err := ParseVersion("2:1.2.3:4-1")
	if c.Check(err, IsNil, Commentf("Got unexpected error %s", err)) == true {
		c.Check(v, DeepEquals, &Version{
			Epoch:           2,
			UpstreamVersion: "1.2.3:4",
			DebianRevision:  "1",
		})
	}

	v, err = ParseVersion("1.2.3:4-1")
	c.Check(v, IsNil)
	c.Check(err, ErrorMatches, "Invalid upstream version `%s', it should not contain an colon since epoch is 0.")
}

func (s *VersionSuite) TestVersionWithNoDebRevisionContainsNoHyphen(c *C) {
	v, err := ParseVersion("1.2.3-4-0+foo")
	if c.Check(err, IsNil, Commentf("Got unexpected error %s", err)) == true {
		c.Check(v, DeepEquals, &Version{
			Epoch:           0,
			UpstreamVersion: "1.2.3-4",
			DebianRevision:  "0+foo",
		})
	}

	v, err = ParseVersion("3:1.2.3-4:3")
	c.Check(v, IsNil)
	c.Check(err, ErrorMatches, "Invalid upstream version `%s', it should not contain an hyphen since debian revision is 0")

}
