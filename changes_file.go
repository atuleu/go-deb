package deb

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

type ChangesFileRef struct {
	Identifier SourcePackageRef
	Suffix     string
}

// Represents a .changes file content
type ChangesFile struct {
	Ref ChangesFileRef

	//The format of the change file itself
	Format     Version
	Date       time.Time
	Arch       string
	Dist       Distribution
	Maintainer *mail.Address

	Description string
	Changes     string

	Md5Files    []FileReference
	Sha1Files   []FileReference
	Sha256Files []FileReference
}

func (c *ChangesFileRef) Filename() string {
	return fmt.Sprintf("%s_%s_%s.changes", c.Identifier.Source, c.Identifier.Ver, c.Suffix)
}

var debFileRx = regexp.MustCompile(`^(.*)_(.*)_(.*).(deb|udeb)$`)

func (c *ChangesFile) BinaryPackages() ([]BinaryPackageRef, error) {
	res := make([]BinaryPackageRef, 0, cap(c.Md5Files))
	for _, f := range c.Md5Files {
		if strings.HasSuffix(f.Name, ".deb") == false &&
			strings.HasSuffix(f.Name, ".udeb") == false {
			continue
		}
		matches := debFileRx.FindStringSubmatch(f.Name)
		if matches == nil {
			return nil, fmt.Errorf(".changes file list invalid file `%s'", f.Name)
		}
		ver, err := ParseVersion(matches[2])
		if err != nil {
			return nil, err
		}
		res = append(res, BinaryPackageRef{
			Name: matches[1],
			Ver:  *ver,
			Arch: Architecture(matches[3]),
		})
	}
	return res, nil
}
