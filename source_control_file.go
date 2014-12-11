package deb

import (
	"fmt"
	"io"
	"net/mail"
	"path"
	"regexp"
)

// Represents a .dsc file content Only Mandatory according to the
// Debian Policy Manual file are represented.
type SourceControlFile struct {
	Identifier SourcePackageRef

	BasePath string
	//A Format for a source file can be 1.0 3.0 (native) or 3.0 (quilt)
	Format string

	// The maintainer email address, which is mandatory
	Maintainer *mail.Address

	// A list of md5 checksumed files
	Md5Files []FileReference
	// A list of sha1 checksumed files
	Sha1Files []FileReference
	// A list of sha256 checksumed files
	Sha256Files []FileReference
}

func (dsc *SourceControlFile) Filename() string {
	return fmt.Sprintf("%s_%s.dsc", dsc.Identifier.Source, dsc.Identifier.Ver)
}

func (dsc *SourceControlFile) ChangesFilename() string {
	return fmt.Sprintf("%s_%s_source.changes", dsc.Identifier.Source, dsc.Identifier.Ver)
}

func IsDscFileName(p string) error {
	rx := regexp.MustCompile(`^(.*)_(.*)\.dsc$`)
	m := rx.FindStringSubmatch(path.Base(p))
	if m == nil {
		return fmt.Errorf("Wrong file name syntax %s", p)
	}
	_, err := ParseVersion(m[2])
	return err
}

type dscFieldParser func(dsc *SourceControlFile, f ControlField) error

func ParseDsc(r io.Reader) (*SourceControlFile, error) {
	res := &SourceControlFile{}
	l := NewControlFileLexer(r)
	called := make(map[string]bool)
	for {
		f, err := l.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf(".dsc parse error: %s", err)
		}

		if IsNewParagraph(f) == true {
			return nil, fmt.Errorf(".dsc parse error:  expect to have one single paragraph")
		}

		fnc, ok := parseDscFunctions[f.Name]
		if ok == false {
			return nil, fmt.Errorf(".dsc parse error: Unknown field %s", f.Name)
		}
		if fnc == nil {
			continue
		}

		err = fnc(res, f)
		if err != nil {
			return nil, fmt.Errorf(".dsc field parse error %s: %s", f, err)
		}
		called[f.Name] = true
	}

	for fName, fnc := range parseDscFunctions {
		_, found := called[fName]
		if found == false && fnc != nil {
			return nil, fmt.Errorf(".dsc parse error, missing mandatory field %s", fName)
		}
	}

	return res, nil
}

func wrapChangeParse(f ControlField, pFn changesFieldParser) (*ChangesFile, error) {
	dummy := &ChangesFile{}
	err := pFn(dummy, f)
	if err != nil {
		return nil, err
	}
	return dummy, nil
}

func parseDscFormat(dsc *SourceControlFile, f ControlField) error {
	if err := expectSingleLine(f); err != nil {
		return err
	}
	switch f.Data[0] {
	case "1.0", "3.0 (native)", "3.0 (quilt)":
		dsc.Format = f.Data[0]
		return nil
	}

	return fmt.Errorf("Invalid format %s", f.Data[0])
}

func parseDscSource(dsc *SourceControlFile, f ControlField) error {
	res, err := wrapChangeParse(f, parseSource)
	if err != nil {
		return err
	}
	dsc.Identifier.Source = res.Ref.Identifier.Source
	return nil
}

func parseDscVersion(dsc *SourceControlFile, f ControlField) error {
	res, err := wrapChangeParse(f, parseVersion)
	if err != nil {
		return err
	}
	dsc.Identifier.Ver = res.Ref.Identifier.Ver
	return nil
}

func parseDscFiles(dsc *SourceControlFile, f ControlField) error {
	res, err := wrapChangeParse(f, parseFiles)
	if err != nil {
		return err
	}
	dsc.Md5Files = res.Md5Files
	return nil
}

func parseDscSha1(dsc *SourceControlFile, f ControlField) error {
	res, err := wrapChangeParse(f, parseSha1)
	if err != nil {
		return err
	}
	dsc.Sha1Files = res.Sha1Files
	return nil
}

func parseDscSha256(dsc *SourceControlFile, f ControlField) error {
	res, err := wrapChangeParse(f, parseSha256)
	if err != nil {
		return err
	}
	dsc.Sha256Files = res.Sha256Files
	return nil
}

func parseDscMaintainer(dsc *SourceControlFile, f ControlField) error {
	res, err := wrapChangeParse(f, parseMaintainer)
	if err != nil {
		return err
	}
	dsc.Maintainer = res.Maintainer
	return nil
}

var parseDscFunctions = map[string]dscFieldParser{
	"Format":                parseDscFormat,
	"Source":                parseDscSource,
	"Binary":                nil,
	"Architecture":          nil,
	"Version":               parseDscVersion,
	"Maintainer":            parseDscMaintainer,
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
	"Checksums-Sha1":        parseDscSha1,
	"Checksums-Sha256":      parseDscSha256,
	"Files":                 parseDscFiles,
}
