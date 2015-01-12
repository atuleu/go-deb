package main

import (
	"regexp"

	deb ".."
)

type AptRepositoryStub struct {
	ArchiveCalled bool
	Err           error
}

func (l *AptRepositoryStub) ArchiveChanges(c *deb.ChangesFile, dir string) error {
	if l.Err != nil {
		return l.Err
	}
	l.ArchiveCalled = true
	return nil
}
func (l *AptRepositoryStub) AddDistribution(deb.Distribution, deb.Architecture) error {
	return nil
}

func (l *AptRepositoryStub) RemoveDistribution(deb.Distribution, deb.Architecture) error {
	return nil
}
func (l *AptRepositoryStub) ListPackage(deb.Distribution, *regexp.Regexp) []deb.BinaryPackageRef {
	return nil

}
func (l *AptRepositoryStub) RemovePackage(deb.Distribution, deb.BinaryPackageRef) error {
	return nil
}

func (l *AptRepositoryStub) Access() AptRepositoryAccess {
	return AptRepositoryAccess{}
}
