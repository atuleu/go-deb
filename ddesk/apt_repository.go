package main

import (
	"io"
	"regexp"

	"golang.org/x/crypto/openpgp"

	deb ".."
)

type AptRepositoryId string

type AptRepositoryAccess struct {
	ID        AptRepositoryId
	Comps     map[deb.Codename]map[deb.Component]bool
	Address   string
	PublicKey *openpgp.Entity
}

type AptRepository interface {
	ArchiveChanges(c *deb.ChangesFile, dir string) error
	AddDistribution(deb.Codename, deb.Architecture) error
	RemoveDistribution(deb.Codename, deb.Architecture) error
	ListPackage(deb.Codename, *regexp.Regexp) []deb.BinaryPackageRef
	RemovePackage(deb.Codename, deb.BinaryPackageRef) error
	Access() *AptRepositoryAccess
}

func NewPPARepositoryAccess(ppaAddress string, enabledDist []deb.Codename) (*AptRepositoryAccess, error) {
	return nil, deb.NotYetImplemented()
}

func NewRemoteAptRepositoryAccess(address string, r io.Reader) (*AptRepositoryAccess, error) {
	return nil, deb.NotYetImplemented()
}
