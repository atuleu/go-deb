package main

import (
	"fmt"
	"io"
	"net/mail"
	"os"
	"path"
	"strings"

	"golang.org/x/crypto/openpgp"

	deb ".."
)

type PgpKeyManager interface {
	Add(r io.Reader) error
	List() openpgp.EntityList
	Remove(string) error
	CheckAndRemoveClearsigned(r io.Reader) (io.Reader, error)
	PrivateShortKeyID() bool
	SetupPrivate() error
}

type AptRepo interface {
	Add(deb.Distribution, deb.Architecture) error
	Remove(deb.Distribution, deb.Architecture) error
	List() map[deb.Distribution][]deb.Architecture
	Include(changes *deb.ChangesFile) error
}

type Interactor struct {
	keyManager PgpKeyManager
	repo       AptRepo
}

func (x *Interactor) AuthorizePublicKey(r io.Reader) error {
	return x.keyManager.Add(r)
}

func (x *Interactor) UnauthorizePublicKey(keyShortId string) error {
	return x.keyManager.Remove(keyShortId)
}

func (x *Interactor) ListAutorizedKeys() openpgp.EntityList {
	return x.keyManager.List()
}

func (x *Interactor) AddDistribution(d deb.Distribution, a deb.Architecture) error {
	return x.repo.Add(d, a)
}

func (x *Interactor) RemoveDistribution(d deb.Distribution, a deb.Architecture) error {
	return x.repo.Remove(d, a)
}

func (x *Interactor) ListDistributions() map[deb.Distribution][]deb.Architecture {
	return x.repo.List()
}

type IncludeResult struct {
	SendTo        *mail.Address
	ShouldReport  bool
	FilesToRemove []string
}

func (x *Interactor) ProcessChangesFile(filepath string) (*IncludeResult, error) {

	res := &IncludeResult{
		SendTo:        nil,
		FilesToRemove: make([]string, 0, 3),
		ShouldReport:  false,
	}

	if strings.HasSuffix(filepath, ".changes") == false {
		return res, fmt.Errorf("Invalid filename %s", filepath)
	}

	f, err := os.Open(filepath)
	if err != nil {
		return res, err
	}

	r, authErr := x.keyManager.CheckAndRemoveClearsigned(f)
	if authErr != nil {
		if r != nil {
			authErr = fmt.Errorf("Unauthorized .changes upload: %s", err)
			res.ShouldReport = true
		} else {
			return res, authErr
		}
	}

	changes, err := deb.ParseChangeFile(r)
	if err != nil {
		return res, err
	}
	res.SendTo = changes.Maintainer
	basepath := path.Dir(filepath)
	for _, f := range changes.Md5Files {
		res.FilesToRemove = append(res.FilesToRemove, path.Join(basepath, f.Name))
	}

	if authErr != nil {
		return res, authErr
	}

	_, ok := x.repo.List()[changes.Dist]
	if ok == false {
		return res, fmt.Errorf("Unsupported distribution %s", changes.Dist)
	}

	return res, x.repo.Include(changes)
}
