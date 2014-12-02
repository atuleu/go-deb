package deb

import (
	"encoding/hex"
	"fmt"
	"io"
	"net/mail"
	"regexp"
	"strconv"
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
	Arch       []Architecture
	Binary     []string
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

type changesFieldParser func(c *ChangesFile, f ControlField) error

func expectSingleLine(f ControlField) error {
	if len(f.Data) != 1 {
		return fmt.Errorf("expected a single line field")
	}
	return nil
}

func expectMultiLine(f ControlField) error {
	if len(f.Data) <= 1 || len(f.Data[0]) != 0 {
		return fmt.Errorf("expected a multi-line field, first line empty")
	}
	return nil
}

func parseFormat(c *ChangesFile, f ControlField) error {
	if err := expectSingleLine(f); err != nil {
		return err
	}
	ver, err := ParseVersion(f.Data[0])
	if err != nil {
		return err
	}

	if ver.Epoch != 0 || ver.DebianRevision != "0" {
		return fmt.Errorf("it should have no epoch or debian revision")
	}
	c.Format = *ver
	return nil
}

func parseDate(c *ChangesFile, f ControlField) error {
	if err := expectSingleLine(f); err != nil {
		return err
	}
	elems := strings.Split(f.Data[0], " ")
	offset := elems[len(elems)-1]
	if len(offset) != 5 || (offset[0] != '+' && offset[0] != '-') {
		return fmt.Errorf("invalid UTC offset `%s'", offset)
	}
	offsetHours, err := strconv.Atoi(offset[0:3])
	if err != nil {
		return fmt.Errorf("invalid UTC offset `%s'", offset)
	}
	offsetMinutes, err := strconv.Atoi(offset[3:5])
	if err != nil {
		return fmt.Errorf("invalid UTC offset `%s'", offset)
	}
	if offsetMinutes < 0 || offsetMinutes > 59 {
		return fmt.Errorf("invalid UTC offset `%s'", offset)
	}
	if offset[0] == '-' {
		offsetMinutes = -offsetMinutes
	}

	// parse the offseted time
	date, err := time.Parse("Mon, 02 Jan 2006 15:04:05",
		strings.Join(elems[0:len(elems)-1], " "))
	if err != nil {
		return err
	}

	//remove the extracted offset
	c.Date = date.Add(-time.Duration(offsetHours)*time.Hour - time.Duration(offsetMinutes)*time.Minute)
	return nil
}

func parseArchitecture(c *ChangesFile, f ControlField) error {
	c.Arch = []Architecture{}
	for _, l := range f.Data {
		for _, as := range strings.Split(l, " ") {
			a := Architecture(as)
			_, ok := ArchitectureList[a]
			if ok == false {
				return fmt.Errorf("unknown architecture %s", a)
			}
			c.Arch = append(c.Arch, a)
		}
	}
	return nil
}

func parseDistribution(c *ChangesFile, f ControlField) error {
	if err := expectSingleLine(f); err != nil {
		return err
	}

	d := strings.TrimSpace(f.Data[0])
	if strings.Contains(d, " ") {
		return fmt.Errorf("does not contains a single distribution")
	}

	c.Dist = Distribution(d)
	return nil
}

func parseMaintainer(c *ChangesFile, f ControlField) error {
	if err := expectSingleLine(f); err != nil {
		return err
	}

	mail, err := mail.ParseAddress(f.Data[0])
	if err != nil {
		return err
	}

	c.Maintainer = mail
	return nil
}

func parseChanges(c *ChangesFile, f ControlField) error {
	if err := expectMultiLine(f); err != nil {
		return err
	}
	c.Changes = strings.Join(f.Data[1:], "\n")
	return nil
}

func parseDescription(c *ChangesFile, f ControlField) error {
	if err := expectMultiLine(f); err != nil {
		return err
	}
	c.Description = strings.Join(f.Data[1:], "\n")
	return nil
}

func parseFileList(f ControlField) ([]FileReference, error) {
	files := []FileReference{}
	if err := expectMultiLine(f); err != nil {
		return nil, err
	}

	for _, line := range f.Data[1:] {
		data := strings.Split(line, " ")
		if len(data) != 3 && len(data) != 5 {
			return nil, fmt.Errorf("invalid line `%s' (%d elements), expected `checksum size [section priority] name'", line, len(data))
		}

		file := FileReference{
			Name: data[len(data)-1],
		}

		_, err := fmt.Sscanf(data[1], "%d", &file.Size)
		if err != nil {
			return nil, err
		}
		file.Checksum, err = hex.DecodeString(data[0])
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func parseSha1(c *ChangesFile, f ControlField) error {
	files, err := parseFileList(f)
	if err != nil {
		return err
	}
	c.Sha1Files = files
	return nil
}

func parseSha256(c *ChangesFile, f ControlField) error {
	files, err := parseFileList(f)
	if err != nil {
		return err
	}
	c.Sha256Files = files
	return nil

}

func parseFiles(c *ChangesFile, f ControlField) error {
	files, err := parseFileList(f)
	if err != nil {
		return err
	}
	c.Md5Files = files
	return nil
}

func parseSource(c *ChangesFile, f ControlField) error {
	if err := expectSingleLine(f); err != nil {
		return err
	}

	s := strings.TrimSpace(f.Data[0])
	if strings.Contains(s, " ") {
		return fmt.Errorf("multiple source name")
	}

	c.Ref.Identifier.Source = s
	return nil
}

func parseVersion(c *ChangesFile, f ControlField) error {
	if err := expectSingleLine(f); err != nil {
		return err
	}
	ver, err := ParseVersion(f.Data[0])
	if err != nil {
		return err
	}
	c.Ref.Identifier.Ver = *ver
	return nil
}

func parseBinary(c *ChangesFile, f ControlField) error {
	c.Binary = []string{}
	for _, line := range f.Data {
		c.Binary = append(c.Binary, strings.Split(line, " ")...)
	}
	return nil
}

var parseFunctions = map[string]changesFieldParser{
	"Format":           parseFormat,
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

func ParseChangeFile(r io.Reader) (*ChangesFile, error) {
	//TODO remove/decrypt signature if needed

	l := NewControlFileLexer(r)

	res := &ChangesFile{}
	found := map[string]bool{}
	for {
		f, err := l.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf(".changes parse error: %s", err)
		}

		if IsNewParagraph(f) == true {
			return nil, fmt.Errorf(".changes are expected to have a single paragraph")
		}

		fnc, ok := parseFunctions[f.Name]
		if ok == false {
			return nil, fmt.Errorf("Unexpected .changes field %s", f.Name)
		}
		if fnc == nil {
			//we ignore the non-mandatory field
			continue
		}
		err = fnc(res, f)
		if err != nil {
			return nil, fmt.Errorf("Invalid .changes field %s: %s", f, err)
		}
		found[f.Name] = true
	}

	for fieldName, fnc := range parseFunctions {
		if fnc == nil {
			// this is a non-mandatory field
			continue
		}
		_, wasFound := found[fieldName]
		if wasFound == false {
			return nil, fmt.Errorf(".changes miss mandatory field %s", fieldName)
		}

	}

	return res, nil
}
