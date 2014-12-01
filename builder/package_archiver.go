package main

import (
	deb ".."
)

type Log string

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
