package main

import (
	"regexp"

	"golang.org/x/crypto/openpgp/packet"

	deb ".."
)

type AptRepositoryAccess struct {
	Dists      []deb.Codename
	Components []deb.Component
	Address    string
	SigningKey *packet.PublicKey
}

type AptRepository interface {
	ArchiveChanges(c *deb.ChangesFile, dir string) error
	AddDistribution(deb.Codename, deb.Architecture) error
	RemoveDistribution(deb.Codename, deb.Architecture) error
	ListPackage(deb.Codename, *regexp.Regexp) []deb.BinaryPackageRef
	RemovePackage(deb.Codename, deb.BinaryPackageRef) error
	Access() AptRepositoryAccess
}
