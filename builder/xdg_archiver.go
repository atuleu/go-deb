package main

import (
	deb ".."

	"github.com/nightlyone/lockfile"
)

type XdgArchiver struct {
	basepath string
	lock     lockfile.Lockfile
}

func NewXdgArchiver(*XdgArchiver, error) {
	return nil, deb.NotYetImplemented()
}

func (a *XdgArchiver) ArchiveSource(p deb.SourceControlFile) (*ArchivedSource, error) {
	return nil, deb.NotYetImplemented()
}

func (a *XdgArchiver) ArchiveBuildResult(b BuildResult) (*BuildResult, error) {
	return nil, deb.NotYetImplemented()
}

func (a *XdgArchiver) GetArchivedSource(p deb.SourcePackageRef) (*ArchivedSource, error) {
	return nil, deb.NotYetImplemented()
}

func (a *XdgArchiver) GetBuildResult(p deb.SourcePackageRef) (*BuildResult, error) {
	return nil, deb.NotYetImplemented()
}
