package main

import (
	"regexp"

	"golang.org/x/crypto/openpgp/packet"

	deb ".."
)

type AptRepositoryID string

type AptRepositoryAccess struct {
	ID         AptRepositoryID
	Components map[deb.Codename][]deb.Component
	Address    string
	SigningKey *packet.PublicKey
}

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
		for c, _ := range set {
			a.Components[d] = append(a.Components[d], c)
		}
	}
}

func (a *AptRepositoryAccess) String() string {
	return string(a.ID)
}

type AptRepository interface {
	ArchiveChanges(c *deb.ChangesFile, dir string) error
	AddDistribution(deb.Codename, deb.Architecture) error
	RemoveDistribution(deb.Codename, deb.Architecture) error
	ListPackage(deb.Codename, *regexp.Regexp) []deb.BinaryPackageRef
	RemovePackage(deb.Codename, deb.BinaryPackageRef) error
	Access() *AptRepositoryAccess
}
