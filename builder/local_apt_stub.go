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
func (l *LocalAptRepositoryStub) AddDistribution(DistributionAndArch) error {
	return nil
}

func (l *LocalAptRepositoryStub) RemoveDistribution(DistributionAndArch) error {
	return nil
}
func (l *LocalAptRepositoryStub) ListPackage(deb.Distribution, *regexp.Regexp) []deb.BinaryPackageRef {
	return nil

}
func (l *LocalAptRepositoryStub) RemovePackage(deb.Distribution, deb.BinaryPackageRef) error {
	return nil
}
