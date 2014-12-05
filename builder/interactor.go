package main

import deb ".."

type Interactor struct {
	archiver        PackageArchiver
	localRepository AptRepository
	builder         DebianBuilder
	history         History
	u               UserDistributionSupport
}

func NewInteractor() (*Interactor, error) {
	return nil, deb.NotYetImplemented()
}
