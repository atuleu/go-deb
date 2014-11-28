package main

import (
	"fmt"
	"io"

	deb "../"
)

type AutobuildSourcePackage struct{}

type Log string

type StagingResult struct {
	//The Reference of the build packet
	Ref  deb.SourcePackageRef
	Dist deb.Distribution
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
	Changes *deb.ChangesFileRef
	// The base path to find all files on the current filesystem
	BasePath string
}

// Interface of a module that can build packages
type DebianBuilder interface {
	BuildPackage(p deb.SourceControlFile, output io.Writer) (*BuildResult, error)
	InitDistribution(DistributionAndArch) error
	UpdateDistribution(DistributionAndArch) error
	AvailableDistributions() []deb.Distribution
	AvailableArchitectures(d deb.Distribution) []deb.Architecture
	CurrentBuild() *InBuildResult
}

type ArchivedSource struct {
	Changes    *deb.ChangesFileRef
	Dsc        deb.SourceControlFile
	TargetDist deb.Distribution
	BasePath   string
}

type PackageArchiver interface {
	//Archive a SourcePackage
	ArchiveSource(p deb.SourceControlFile) (*ArchivedSource, error)
	//Archive a BuildResult
	ArchiveBuildResult(b BuildResult) (*BuildResult, error)
	// Returns the archived source package
	GetArchivedSource(p deb.SourcePackageRef) (*ArchivedSource, error)
	//Returns the archived build result
	GetBuildResult(p deb.SourcePackageRef) (*BuildResult, error)
}

type History interface {
	Append(deb.SourcePackageRef)
	Get() []deb.SourcePackageRef
	RemoveFront(deb.SourcePackageRef)
}

type LocalAptRepository interface {
	ArchiveBuiltResult(BuildResult) error
}

type Interactor struct {
	p PackageArchiver
	b DebianBuilder
	h History
}

// Builds a deb.SourcePackage and return the result. If a io.Writer is
// passed, the build process output will be copied to it.
func (x *Interactor) BuildPackage(s deb.SourceControlFile, buildOut io.Writer) (*BuildResult, error) {
	a, err := x.p.ArchiveSource(s)
	if err != nil {
		return nil, fmt.Errorf("Could not archive source package `%s': %s", s.Identifier, err)
	}

	supportsTarget := false
	for _, d := range x.b.AvailableDistributions() {
		if d == a.TargetDist {
			supportsTarget = true
			break
		}
	}

	if supportsTarget == false {
		return nil, fmt.Errorf("Target distribution `%s' of source package `%s' is not supported", a.TargetDist, s.Identifier)
	}

	buildRes, err := x.b.BuildPackage(a.Dsc, buildOut)

	buildRes, archErr := x.p.ArchiveBuildResult(*buildRes)

	if archErr != nil {
		x.h.RemoveFront(s.Identifier)
		return nil, fmt.Errorf("Failed to archive build result of `%s': %s", s.Identifier, archErr)
	}

	if err == nil {
		x.h.Append(s.Identifier)
	}

	return buildRes, err
}

// Builds debian package from a Debianized Git repository.
func (x *Interactor) BuildDebianizedGit(path string, buildOut io.Writer) (*BuildResult, error) {
	return nil, deb.NotYetImplemented()
}

//Builds a package from an autobuild ( http://github.com/jessevdk/autobuild ) source package.
func (x *Interactor) BuildAutobuildSource(p AutobuildSourcePackage, buildOut io.Writer) (*BuildResult, error) {
	return nil, deb.NotYetImplemented()
}

// Returns the current running build status or nil if none is currently build
func (x *Interactor) GetCurrentBuild() *InBuildResult {
	return nil
}

// Returns the build result of the last built of the given source package
func (x *Interactor) GetBuildResult(s deb.SourcePackageRef) (*BuildResult, error) {
	return x.p.GetBuildResult(s)
}

// Returns the deb.SourcePackageRef of the last succesfull build of the package user
func (x *Interactor) GetLastSuccesfullUserBuild() *deb.SourcePackageRef {
	successfulls := x.h.Get()
	if len(successfulls) == 0 {
		return nil
	}
	return &(successfulls[0])
}
