package deb

import (
	"fmt"
	"path"
	"regexp"
)

type Distribution string

type Component string

type Architecture string

const (
	Any    Architecture = "any"
	All    Architecture = "all"
	Amd64  Architecture = "amd64"
	I386   Architecture = "i386"
	Source Architecture = "source"
	Armel  Architecture = "armel"
)

var ArchitectureList = map[Architecture]bool{
	Any:    true,
	All:    true,
	Amd64:  true,
	I386:   true,
	Source: true,
	Armel:  true,
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
	Size     uint
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

func (f *FileReference) CheckFile(basepath string) error {
	return NotYetImplemented()
}
