package deb

import (
	"fmt"
	"io"
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
	Source     string
	Ver        Version `field:"Version"`
	Format     Version
	Date       time.Time
	Arch       []Architecture `field:"Architectures"`
	Binary     []string
	Dist       Distribution `field:"Distribution"`
	Maintainer *mail.Address

	Description string
	Changes     string

	Md5Files    []FileReference `field:"Files"`
	Sha1Files   []FileReference `field:"ChecksumsSha1"`
	Sha256Files []FileReference `field:"ChecksumsSha256"`
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

var changesParseFunctions = map[string]controlFieldParser{
	"Format":           parseChangesFormat,
	"Date":             parseDate,
	"Source":           parseSource,
	"Binary":           parseBinary,
	"Architecture":     parseArchitecture,
	"Version":          parseVersion,
	"Distribution":     parseDistribution,
	"Urgency":          nil,
	"Changed-By":       nil,
	"Description":      parseDescription,
	"Closes":           nil,
	"Changes":          parseChanges,
	"Checksums-Sha1":   parseSha1,
	"Checksums-Sha256": parseSha256,
	"Files":            parseFiles,
	"Maintainer":       parseMaintainer,
}

//Parses an unsigned .changes file
func ParseChangeFile(r io.Reader) (*ChangesFile, error) {
	p := controlFileParser{
		l:        NewControlFileLexer(r),
		fMapper:  changesParseFunctions,
		required: make([]string, 0),
	}
	for k, v := range p.fMapper {
		if v != nil {
			p.required = append(p.required, k)
		}
	}

	res := &ChangesFile{}
	err := p.parse(res)
	if err != nil {
		return nil, fmt.Errorf(".changes parse error: %s", err)
	}
	res.Ref.Identifier.Source = res.Source
	res.Ref.Identifier.Ver = res.Ver
	return res, nil
}
