package deb

import (
	"fmt"
	"io"
	"net/mail"
	"path"
	"regexp"
)

// SourceControlFile represents a .dsc file content.
//
// Only Mandatory according to the Debian Policy Manual file are
// represented.
type SourceControlFile struct {
	// need to repeat here for parsing
	Source string  `field:"Source"`
	Ver    Version `field:"Version"`

	Identifier SourcePackageRef

	BasePath string
	//A Format for a source file can be 1.0 3.0 (native) or 3.0 (quilt)
	Format string

	Archs []Architecture `field:"Architectures"`

	// The maintainer email address, which is mandatory
	Maintainer *mail.Address

	// A list of md5 checksumed files
	Md5Files []FileReference `field:"Files"`
	// A list of sha1 checksumed files
	Sha1Files []FileReference `field:"ChecksumsSha1"`
	// A list of sha256 checksumed files
	Sha256Files []FileReference `field:"ChecksumsSha256"`
}

// Filename is the expected name a .dsc file should have
func (dsc *SourceControlFile) Filename() string {
	return fmt.Sprintf("%s_%s.dsc", dsc.Identifier.Source, dsc.Identifier.Ver)
}

// ChangesFilename is the expected name a .changes file should have
// for a source upload only.
func (dsc *SourceControlFile) ChangesFilename() string {
	return fmt.Sprintf("%s_%s_source.changes", dsc.Identifier.Source, dsc.Identifier.Ver)
}

// IsDscFileName checks if the given file basename is a filename
// expected for a .dsc
func IsDscFileName(p string) error {
	rx := regexp.MustCompile(`^(.*)_(.*)\.dsc$`)
	p = path.Base(p)
	m := rx.FindStringSubmatch(p)
	if m == nil {
		return fmt.Errorf("Wrong filename syntax %s", p)
	}
	_, err := ParseVersion(m[2])
	return err
}

// ParseDsc parses the content of a reader and return its content or
// an error.
func ParseDsc(r io.Reader) (*SourceControlFile, error) {
	p := controlFileParser{
		l:        NewControlFileLexer(r),
		fMapper:  dscParsers,
		required: make([]string, 0),
	}
	for k, v := range p.fMapper {
		if k == "Architecture" {
			//we need this, but this is not mandatory according to debian policy
			continue
		}
		if v != nil {
			p.required = append(p.required, k)
		}
	}

	res := &SourceControlFile{}
	err := p.parse(res)
	if err != nil {
		return nil, fmt.Errorf(".dsc parse error: %s", err)
	}

	res.Identifier.Source = res.Source
	res.Identifier.Ver = res.Ver

	return res, nil

}

func parseDscFormat(f ControlField, v interface{}) error {
	if err := expectSingleLine(f); err != nil {
		return err
	}
	switch f.Data[0] {
	case "1.0", "3.0 (native)", "3.0 (quilt)":
		return setField(v, "Format", f.Data[0])
	}

	return fmt.Errorf("invalid format %s", f.Data[0])
}

var dscParsers = map[string]controlFieldParser{
	"Format":                parseDscFormat,
	"Source":                parseSource,
	"Binary":                nil,
	"Architecture":          parseArchitecture,
	"Version":               parseVersion,
	"Maintainer":            parseMaintainer,
	"Uploaders":             nil,
	"Homepage":              nil,
	"Vcs-Browser":           nil,
	"Vcs-Arch":              nil,
	"Vcs-Bzr":               nil,
	"Vcs-Cvs":               nil,
	"Vcs-Darcs":             nil,
	"Vcs-Git":               nil,
	"Vcs-Hg":                nil,
	"Vcs-Mtn":               nil,
	"Vcs-Svn":               nil,
	"Dgit":                  nil,
	"Standards-Version":     nil,
	"Build-Depends":         nil,
	"Build-Depends-Indep":   nil,
	"Build-Conflicts":       nil,
	"Build-Conflicts-Indep": nil,
	"Package-List":          nil,
	"Checksums-Sha1":        parseSha1,
	"Checksums-Sha256":      parseSha256,
	"Files":                 parseFiles,
}
