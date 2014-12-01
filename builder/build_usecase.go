package main

import (
	"fmt"
	"io"
	"regexp"

	deb "../"
)

type AutobuildSourcePackage struct{}

type History interface {
	Append(deb.SourcePackageRef)
	Get() []deb.SourcePackageRef
	RemoveFront(deb.SourcePackageRef)
}

type LocalAptRepository interface {
	ArchiveBuildResult(b *BuildResult) error
	AddDistribution(DistributionAndArch) error
	RemoveDistribution(DistributionAndArch) error
	ListPackage(deb.Distribution, *regexp.Regexp) []deb.BinaryPackageRef
	RemovePackage(deb.Distribution, deb.BinaryPackageRef) error
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

	if archErr == nil {
		archErr = x.a.ArchiveBuildResult(buildRes)
	}

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
