package deb

import (
	"fmt"
	"hash"
	"io"
	"os"
	"path"
	"regexp"
)

// Vendor is providing distributions
type Vendor string

// Codename identifies a distribution.
//
// A distribution is a coherent tree of Pacakge that are built one of
// top of other.
type Codename string

// Component is a component of a distribution
type Component string

// Architecture designate a processor architecture
type Architecture string

const (
	// Any represents any supported architecture
	Any Architecture = "any"
	// All designate binary independant package
	All Architecture = "all"
	// Amd64 represents X86 64 bits processors
	Amd64 Architecture = "amd64"
	// I386 represents X86 32 bits processors
	I386 Architecture = "i386"
	// Source is not an actual architecture, it designate a source package
	Source Architecture = "source"
	// Armel represents ARM little-endian processors
	Armel Architecture = "armel"
)

// Vendor for debian based distribution
const (
	Ubuntu Vendor = "ubuntu"
	Debian Vendor = "debian"
)

// Ubuntu codenames
const (
	Lucid    Codename = "lucid"
	Maverick Codename = "maverick"
	Natty    Codename = "natty"
	Oneiric  Codename = "oneiric"
	Precise  Codename = "precise"
	Quantal  Codename = "quantal"
	Raring   Codename = "raring"
	Trusty   Codename = "trusty"
	Utopic   Codename = "utopic"
	Vivid    Codename = "vivid"
)

// Debian codenames
const (
	Sid      Codename = "sid"
	Squeeze  Codename = "squeeze"
	Wheezy   Codename = "wheezy"
	Jessie   Codename = "jessie"
	Stretch  Codename = "stretch"
	Buster   Codename = "buster"
	Unstable Codename = "unstable"
	Testing  Codename = "testing"
	Stable   Codename = "stable"
)

// ArchitectureList is a set of all existing Architecture
var ArchitectureList = map[Architecture]bool{
	Any:    true,
	All:    true,
	Amd64:  true,
	I386:   true,
	Source: true,
	Armel:  true,
}

// CodenameList maps Codename to their Vendor
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

// SourcePackageRef is a reference to a source package, used to build
// binary package
type SourcePackageRef struct {
	Source string
	Ver    Version
}

// BinaryPackageRef is a reference to a binary package.
type BinaryPackageRef struct {
	Name string
	Ver  Version
	Arch Architecture
}

// FileReference is a reference to an actual file.
type FileReference struct {
	Checksum []byte
	Size     int64
	Name     string
}

func (s SourcePackageRef) String() string {
	return fmt.Sprintf("%s_%s", s.Source, s.Ver)
}

var sourceNameRx = regexp.MustCompile(`(.*)_(.*)\.(debian\.tar\.gz|dsc)`)

// NewRefFromFileName gives a SourcePackageRef from a filename.
//
// It can fails if the name is not correctly formated.
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

// CheckFile test if the file designated by the FileReference has the
// right hash.
//
// It will compute the checksum of the file found in basepath with
// h. It will return an error if it could not be opened or the
// checksum does not match.
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

// ParseArchitecture returns an existing Architecture from a string.
// It will return an error if the Architecture is unknown
func ParseArchitecture(s string) (Architecture, error) {
	a := Architecture(s)
	_, ok := ArchitectureList[a]
	if ok == false {
		return "", fmt.Errorf("Unknown architecture %s", s)
	}
	return a, nil
}
