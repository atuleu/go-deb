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
func (l *AptRepositoryStub) AddDistribution(deb.Codename, deb.Architecture) error {
	return nil
}

func (l *AptRepositoryStub) RemoveDistribution(deb.Codename, deb.Architecture) error {
	return nil
}
func (l *AptRepositoryStub) ListPackage(deb.Codename, *regexp.Regexp) []deb.BinaryPackageRef {
	return nil

}
func (l *AptRepositoryStub) RemovePackage(deb.Codename, deb.BinaryPackageRef) error {
	return nil
}

func (l *AptRepositoryStub) Access() AptRepositoryAccess {
	return nil
}
