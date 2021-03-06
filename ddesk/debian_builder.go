package main

import (
	"io"

	deb ".."
)

// Current status of a debian Package build
type InBuildResult struct {
	// Reference of the source package beeing build
	Ref *deb.SourcePackageRef
	// a byte reader of the current build output
	BuildLog io.Reader
}

// Result of a debian Package build
type BuildResult struct {
	// The log of the build
	BuildLog Log
	// The parsed debian .changes files for built binaries
	Changes *deb.ChangesFile
	// The name of the ChangeFile, relative to BasePath
	ChangesPath string
	// The base path to find all files on the current filesystem
	BasePath string
}

type BuildArguments struct {
	SourcePackage deb.SourceControlFile
	Dist          deb.Codename
	Archs         []deb.Architecture
	Deps          []*AptRepositoryAccess
	Dest          string
}

// Interface of a module that can build packages
type DebianBuilder interface {
	BuildPackage(b BuildArguments, output io.Writer) (*BuildResult, error)
	InitDistribution(d deb.Codename, a deb.Architecture, output io.Writer) error
	RemoveDistribution(d deb.Codename, a deb.Architecture) error
	UpdateDistribution(d deb.Codename, a deb.Architecture, output io.Writer) error
	AvailableDistributions() []deb.Codename
	AvailableArchitectures(d deb.Codename) ArchitectureList
}
