package main

import (
	"regexp"

	deb ".."
)

// AptRepositoryID uniquely designate an external apt repository
type AptRepositoryID string

// AptRepositoryAccess is all the necessary information for apt to
// fetch package from a remote apt repository.
type AptRepositoryAccess struct {
	ID               AptRepositoryID
	Components       map[deb.Codename][]deb.Component
	Address          string
	ArmoredPublicKey []byte
}

// CleanUp is modifying an AptRepositoryAccess to removes duplicate
// deb.Component entries.
func (a *AptRepositoryAccess) CleanUp() {
	for d, comps := range a.Components {
		set := make(map[deb.Component]bool)
		for _, c := range comps {
			set[c] = true
		}
		if len(set) == 0 {
			delete(a.Components, d)
			continue
		}
		a.Components[d] = make([]deb.Component, 0, len(set))
		for c := range set {
			a.Components[d] = append(a.Components[d], c)
		}
	}
}

func (a *AptRepositoryAccess) String() string {
	return string(a.ID)
}

// AptRepository is a collection of packages for a distribution, taht
// can be pulled by the apt-get tool.
type AptRepository interface {
	ArchiveChanges(c *deb.ChangesFile, dir string) error
	AddDistribution(deb.Codename, deb.Architecture) error
	RemoveDistribution(deb.Codename, deb.Architecture) error
	ListPackage(deb.Codename, *regexp.Regexp) []deb.BinaryPackageRef
	RemovePackage(deb.Codename, deb.BinaryPackageRef) error
	Access() *AptRepositoryAccess
}
