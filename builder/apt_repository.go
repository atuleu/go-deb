package main

import (
	"regexp"

	"golang.org/x/crypto/openpgp/packet"

	deb ".."
)

type AptRepositoryAccess struct {
	Dist       deb.Distribution
	Component  deb.Component
	Address    string
	SigningKey *packet.PublicKey
}

type AptRepository interface {
	ArchiveChanges(c *deb.ChangesFile, dir string) error
	AddDistribution(deb.Distribution, deb.Architecture) error
	RemoveDistribution(deb.Distribution, deb.Architecture) error
	ListPackage(deb.Distribution, *regexp.Regexp) []deb.BinaryPackageRef
	RemovePackage(deb.Distribution, deb.BinaryPackageRef) error
	Access() AptRepositoryAccess
}
