package main

import (
	"regexp"

	deb ".."
)

type aptRepositoryStub struct {
	ArchiveCalled bool
	Err           error
}

func (l *aptRepositoryStub) ArchiveChanges(c *deb.ChangesFile, dir string) error {
	if l.Err != nil {
		return l.Err
	}
	l.ArchiveCalled = true
	return nil
}
func (l *aptRepositoryStub) AddDistribution(deb.Codename, deb.Architecture) error {
	return nil
}

func (l *aptRepositoryStub) RemoveDistribution(deb.Codename, deb.Architecture) error {
	return nil
}
func (l *aptRepositoryStub) ListPackage(deb.Codename, *regexp.Regexp) []deb.BinaryPackageRef {
	return nil

}
func (l *aptRepositoryStub) RemovePackage(deb.Codename, deb.BinaryPackageRef) error {
	return nil
}

func (l *aptRepositoryStub) Access() *AptRepositoryAccess {
	return nil
}
