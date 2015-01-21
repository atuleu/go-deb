package main

import (
	"io"
	"regexp"

	"golang.org/x/crypto/openpgp"

	deb ".."
)

type AptRepositoryId string

type AptRepositoryAccess interface {
	ID() AptRepositoryId
	Components(deb.Codename) []deb.Component
	AptURL() string
	PublicKey() *openpgp.Entity
}

type AptRepository interface {
	ArchiveChanges(c *deb.ChangesFile, dir string) error
	AddDistribution(deb.Codename, deb.Architecture) error
	RemoveDistribution(deb.Codename, deb.Architecture) error
	ListPackage(deb.Codename, *regexp.Regexp) []deb.BinaryPackageRef
	RemovePackage(deb.Codename, deb.BinaryPackageRef) error
	Access() AptRepositoryAccess
}

type PPARepositoryAccess struct {
}

func NewPPARepositoryAccess(ppaAddress string) (*PPARepositoryAccess, error) {
	return nil, deb.NotYetImplemented()
}

type RemoteAptRepositoryAccess struct {
}

func NewRemoteAptRepositoryAccess(address string, r io.Reader) (*AptRepositoryAccess, error) {
	return nil, deb.NotYetImplemented()
}
