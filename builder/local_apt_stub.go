package main

import (
	"regexp"

	deb ".."
)

type LocalAptRepositoryStub struct {
	ArchiveCalled bool
	Err           error
}

func (l *LocalAptRepositoryStub) ArchiveChanges(c *deb.ChangesFile) error {
	if l.Err != nil {
		return l.Err
	}
	l.ArchiveCalled = true
	return nil
}
func (l *LocalAptRepositoryStub) AddDistribution(deb.Distribution, deb.Architecture) error {
	return nil
}

func (l *LocalAptRepositoryStub) RemoveDistribution(deb.Distribution, deb.Architecture) error {
	return nil
}
func (l *LocalAptRepositoryStub) ListPackage(deb.Distribution, *regexp.Regexp) []deb.BinaryPackageRef {
	return nil

}
func (l *LocalAptRepositoryStub) RemovePackage(deb.Distribution, deb.BinaryPackageRef) error {
	return nil
}

func (l *LocalAptRepositoryStub) Access() AptRepositoryAccess {
	return AptRepositoryAccess{}
}
