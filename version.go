package deb

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Represents a version in the debian package management system. All
// informationa are based on the Debian Policy Manual
type Version struct {
	//It is a single small unsigned integer, default to zero
	Epoch uint32

	//The upstream version, should never be empty. starts by a number
	//and contains only [0-a-ZA-Z\.+~] and optionally : if epoch is
	//zero and - if there is no debian revision
	UpstreamVersion string

	//The debian version, could be omitted. contains only alphanum and
	//+ - ~ if 0 it can be ommited
	DebianRevision string
}

func ParseVersion(s string) (*Version, error) {
	epoch := 0
	epochMatches := epochRx.FindStringSubmatch(s)
	if epochMatches != nil {
		// here epochMatches[1] matches regexp [1-9][0-9]*, so it is
		// convertible !
		epoch, _ = strconv.Atoi(epochMatches[1])
		s = strings.TrimPrefix(s, epochMatches[0])
	}

	res := &Version{
		Epoch:          uint32(epoch),
		DebianRevision: "0",
	}

	revMatches := debRevRx.FindStringSubmatch(s)
	if revMatches != nil {
		res.DebianRevision = revMatches[1]
		s = strings.TrimSuffix(s, revMatches[0])
	}

	if upVerRx.MatchString(s) == false {
		return nil, fmt.Errorf("Invalid upstream version syntax `%s'", s)
	}

	if res.Epoch == 0 && strings.Contains(s, ":") {
		return nil, fmt.Errorf("Invalid upstream version `%s', it should not contain a colon since epoch is 0", s)
	}

	if res.DebianRevision == "0" && strings.Contains(s, "-") {
		return nil, fmt.Errorf("Invalid upstream version `%s', it should not contain an hyphen since debian revision is 0", s)
	}

	res.UpstreamVersion = s
	return res, nil
}

var epochRx = regexp.MustCompile(`^([1-9][0-9]*):`)
var debRevRx = regexp.MustCompile(`-([0-9a-zA-Z\+~\.]+)$`)
var upVerRx = regexp.MustCompile(`^[0-9][0-9a-zA-Z\.\+~:\-]*$`)

func (v Version) String() string {
	prefix := ""
	if v.Epoch != 0 {
		prefix = fmt.Sprintf("%d:", v.Epoch)
	}
	suffix := ""
	if v.DebianRevision != "0" {
		suffix = "-" + v.DebianRevision
	}

	return prefix + v.UpstreamVersion + suffix
}
