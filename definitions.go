package deb

import "fmt"

type Distribution string

type Component string

type Architecture string

const (
	Any   Architecture = "any"
	All   Architecture = "all"
	Amd64 Architecture = "amd64"
	I386  Architecture = "i386"
	Armel Architecture = "armel"
)

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
