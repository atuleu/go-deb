package deb

import (
	"fmt"
	"hash"
	"io"
	"os"
	"path"
	"regexp"
)

type Vendor string

type Codename string

type Component string

type Architecture string

const (
	Any      Architecture = "any"
	All      Architecture = "all"
	Amd64    Architecture = "amd64"
	I386     Architecture = "i386"
	Source   Architecture = "source"
	Armel    Architecture = "armel"
	Ubuntu   Vendor       = "ubuntu"
	Debian   Vendor       = "debian"
	Lucid    Codename     = "lucid"
	Maverick Codename     = "maverick"
	Natty    Codename     = "natty"
	Oneiric  Codename     = "oneiric"
	Precise  Codename     = "precise"
	Quantal  Codename     = "quantal"
	Raring   Codename     = "raring"
	Trusty   Codename     = "trusty"
	Utopic   Codename     = "utopic"
	Vivid    Codename     = "vivid"
	Sid      Codename     = "sid"
	Squeeze  Codename     = "squeeze"
	Wheezy   Codename     = "wheezy"
	Jessie   Codename     = "jessie"
	Stretch  Codename     = "stretch"
	Buster   Codename     = "buster"
	Unstable Codename     = "unstable"
	Testing  Codename     = "testing"
	Stable   Codename     = "stable"
)

var ArchitectureList = map[Architecture]bool{
	Any:    true,
	All:    true,
	Amd64:  true,
	I386:   true,
	Source: true,
	Armel:  true,
}

var CodenameList = map[Codename]Vendor{
	Sid:      Debian,
	Squeeze:  Debian,
	Wheezy:   Debian,
	Jessie:   Debian,
	Stretch:  Debian,
	Buster:   Debian,
	Unstable: Debian,
	Testing:  Debian,
	Stable:   Debian,
	Lucid:    Ubuntu,
	Maverick: Ubuntu,
	Natty:    Ubuntu,
	Oneiric:  Ubuntu,
	Precise:  Ubuntu,
	Quantal:  Ubuntu,
	Raring:   Ubuntu,
	Trusty:   Ubuntu,
	Utopic:   Ubuntu,
	Vivid:    Ubuntu,
}

type SourcePackageRef struct {
	Source string
	Ver    Version
}

type BinaryPackageRef struct {
	Name string
	Ver  Version
	Arch Architecture
}

type FileReference struct {
	Checksum []byte
	Size     int64
	Name     string
}

func (s SourcePackageRef) String() string {
	return fmt.Sprintf("%s_%s", s.Source, s.Ver)
}

var sourceNameRx *regexp.Regexp = regexp.MustCompile(`(.*)_(.*)\.(debian\.tar\.gz|dsc)`)

func NewRefFromFileName(p string) (*SourcePackageRef, error) {
	matches := sourceNameRx.FindStringSubmatch(path.Base(p))
	if matches == nil {
		return nil, fmt.Errorf("Invalid file name %s", p)
	}
	ver, err := ParseVersion(matches[2])
	if err != nil {
		return nil, err
	}
	return &SourcePackageRef{
		Source: matches[1],
		Ver:    *ver,
	}, nil
}

func (f *FileReference) CheckFile(basepath string, h hash.Hash) error {
	fPath := path.Join(basepath, f.Name)
	fi, err := os.Stat(fPath)
	if err != nil {
		return err
	}
	if fi.Size() != f.Size {
		return fmt.Errorf("Wrong file size %d, expected %d", fi.Size(), f.Size)
	}

	file, err := os.Open(fPath)
	if err != nil {
		return err
	}
	_, err = io.Copy(h, file)
	cs := h.Sum(nil)
	err = fmt.Errorf("Mismatched checksum %x, expected %x", cs, f.Checksum)
	if len(cs) != len(f.Checksum) {
		return err
	}
	for idx, b := range f.Checksum {
		if cs[idx] != b {
			return err
		}
	}
	return nil
}
