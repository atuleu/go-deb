package main

import (
	"io"

	deb ".."
)

type DistributionAndArch struct {
	Dist deb.Distribution
	Arch deb.Architecture
}

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
	// The base path to find all files on the current filesystem
	BasePath string
}

// Interface of a module that can build packages
type DebianBuilder interface {
	BuildPackage(p deb.SourceControlFile, output io.Writer) (*BuildResult, error)
	InitDistribution(d DistributionAndArch, output io.Writer) error
	RemoveDistribution(DistributionAndArch) error
	UpdateDistribution(DistributionAndArch) error
	AvailableDistributions() []deb.Distribution
	AvailableArchitectures(d deb.Distribution) []deb.Architecture
}
