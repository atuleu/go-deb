package main

import deb ".."

type DistributionInitResult struct{}

type DistributionAndArch struct {
	Dist deb.Distribution
	Arch deb.Architecture
}

func (x *Interactor) AddDistributionSupport(d deb.Distribution, a deb.Architecture) (*DistributionInitResult, error) {
	return nil, deb.NotYetImplemented()
}

func (x *Interactor) RemoveDistributionSupport(d deb.Distribution, a deb.Architecture, clearCache bool) error {
	return deb.NotYetImplemented()
}

func (x *Interactor) GetSupportedDistribution() []DistributionAndArch {
	return nil
}

func (x *Interactor) UpdateDistribution(DistributionAndArch) error {
	return deb.NotYetImplemented()
}

func (x *Interactor) ListBuildPackage(d deb.Distribution) []deb.SourcePackageRef {
	return nil
}
