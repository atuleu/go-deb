package main

import deb ".."

type Interactor struct {
	p PackageArchiver
	a LocalAptRepository
	b DebianBuilder
	h History
	u UserDistributionSupport
}

func NewInteractor() (*Interactor, error) {
	return nil, deb.NotYetImplemented()
}
