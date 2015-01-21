package main

import (
	"io"

	deb ".."
)

type AptDependencyManager interface {
	Store(AptRepositoryAccess) error
	Remove(AptRepositoryId) error
	List() map[AptRepositoryId]AptRepositoryAccess
}

func (x *Interactor) AddAptDependency(address string, dists []deb.Codename, comps []deb.Component, key io.Reader) error {
	return deb.NotYetImplemented()
}

func (x *Interactor) AddPPADependency(id string) error {
	return deb.NotYetImplemented()
}

func (x *Interactor) RemoveAptDependency(id string, dist []deb.Codename, comps []deb.Component) error {
	return deb.NotYetImplemented()
}
