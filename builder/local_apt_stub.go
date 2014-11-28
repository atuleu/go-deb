package main

import (
	"regexp"

	deb ".."
)

type LocalAptRepositoryStub struct {
	ArchiveCalled bool
	Err           error
}

func (l *LocalAptRepositoryStub) ArchiveBuildResult(b *BuildResult) error {
	if l.Err != nil {
		return l.Err
	}
	l.ArchiveCalled = true
	return nil
}
func (l *LocalAptRepositoryStub) AddDistribution(deb.Distribution) error {
	return nil
}
func (l *LocalAptRepositoryStub) RemoveDistribution(deb.Distribution) error {
	return nil
}
func (l *LocalAptRepositoryStub) ListPackage(*regexp.Regexp) []deb.BinaryPackageRef {
	return nil

}
func (l *LocalAptRepositoryStub) RemovePackage(deb.Distribution, deb.BinaryPackageRef) error {
	return nil
}
