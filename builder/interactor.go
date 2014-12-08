package main

import deb ".."

type Interactor struct {
	archiver        PackageArchiver
	localRepository AptRepository
	builder         DebianBuilder
	history         History
	u               UserDistributionSupport
}

func NewInteractor(o *Options) (*Interactor, error) {
	return nil, deb.NotYetImplemented()
}
