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
	Dist          deb.Distribution
	Archs         []deb.Architecture
	Deps          []AptRepositoryAccess
}

// Interface of a module that can build packages
type DebianBuilder interface {
	BuildPackage(b BuildArguments, output io.Writer) (*BuildResult, error)
	InitDistribution(d deb.Distribution, a deb.Architecture, output io.Writer) error
	RemoveDistribution(d deb.Distribution, a deb.Architecture) error
	UpdateDistribution(d deb.Distribution, a deb.Architecture, output io.Writer) error
	AvailableDistributions() []deb.Distribution
	AvailableArchitectures(d deb.Distribution) []deb.Architecture
}
