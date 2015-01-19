package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	deb "../"
)

type AutobuildSourcePackage struct{}

type History interface {
	Append(deb.SourcePackageRef)
	Get() []deb.SourcePackageRef
	RemoveFront(deb.SourcePackageRef)
}

// Builds a deb.SourcePackage and return the result. If a io.Writer is
// passed, the build process output will be copied to it.
func (x *Interactor) BuildPackage(s deb.SourceControlFile, buildOut io.Writer) (*BuildResult, error) {
	a, err := x.archiver.ArchiveSource(s)
	if err != nil {
		return nil, fmt.Errorf("Could not archive source package `%s': %s", s.Identifier, err)
	}

	targetDist := a.Changes.Dist

	supported := x.userDistConfig.Supported()
	archs, ok := supported[targetDist]
	if ok == false || len(archs) == 0 {
		return nil, fmt.Errorf("Target distribution `%s' of source package `%s' is not supported", targetDist, s.Identifier)
	}

	for _, targetArch := range archs {
		found := false
		for _, arch := range x.builder.AvailableArchitectures(targetDist) {
			if arch == targetArch {
				found = true
				break
			}
		}
		if found == false {
			return nil, fmt.Errorf("System consistency error: builder does not support %s-%s", targetDist, targetArch)
		}
	}

	//outputs everything in a temporary directory
	dest, err := ioutil.TempDir("", "go-deb.ddesk_output_")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dest)

	//we copy dsc and source file there
	dsc := a.Dsc
	dsc.BasePath = dest
	files := make([]string, 0, len(dsc.Md5Files)+1)
	for _, f := range a.Dsc.Md5Files {
		files = append(files, f.Name)
	}
	files = append(files, dsc.Filename())

	for _, fPath := range files {
		inPath := path.Join(a.Dsc.BasePath, fPath)
		outPath := path.Join(dsc.BasePath, fPath)
		inF, err := os.Open(inPath)
		if err != nil {
			return nil, err
		}
		outF, err := os.Create(outPath)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(outF, inF)
		if err != nil {
			return nil, err
		}
	}

	//we do the build
	buildRes, err := x.builder.BuildPackage(BuildArguments{
		SourcePackage: dsc,
		Dist:          targetDist,
		Archs:         archs,
		Deps:          []AptRepositoryAccess{x.localRepository.Access()},
		Dest:          dest,
	}, buildOut)
	var archErr error = nil
	if buildRes != nil {
		buildRes, archErr = x.archiver.ArchiveBuildResult(*buildRes)
	}

	if archErr == nil && buildRes != nil {
		archErr = x.localRepository.ArchiveChanges(buildRes.Changes, buildRes.BasePath)
	}

	if archErr != nil {
		x.history.RemoveFront(s.Identifier)
		return nil, fmt.Errorf("Failed to archive build result of `%s': %s", s.Identifier, archErr)
	}

	if err == nil {
		x.history.Append(s.Identifier)
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

// Returns the build result of the last built of the given source package
func (x *Interactor) GetBuildResult(s deb.SourcePackageRef) (*BuildResult, error) {
	return x.archiver.GetBuildResult(s)
}

// Returns the deb.SourcePackageRef of the last succesfull build of the package user
func (x *Interactor) GetLastSuccesfullUserBuild() *deb.SourcePackageRef {
	successfulls := x.history.Get()
	if len(successfulls) == 0 {
		return nil
	}
	return &(successfulls[0])
}
