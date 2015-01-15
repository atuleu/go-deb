package main

import (
	"fmt"
	"io"
	"net/mail"
	"os"
	"path"
	"strings"

	deb ".."
)

func (x *Interactor) AuthorizePublicKey(r io.Reader) error {
	return x.keyManager.Add(r)
}

func (x *Interactor) UnauthorizePublicKey(keyShortId string) error {
	return x.keyManager.Remove(keyShortId)
}

type KeyDescription struct {
	Fullname string
	Id       string
}

func (x *Interactor) ListAutorizedKeys() []KeyDescription {
	res := []KeyDescription{}
	for _, e := range x.keyManager.List() {
		fullname := ""
		for k, _ := range e.Identities {
			fullname = k
			break
		}

		res = append(res,
			KeyDescription{
				Fullname: fullname,
				Id:       e.PrimaryKey.KeyIdShortString(),
			})

	}
	return res

}

func (x *Interactor) AddDistribution(d deb.Codename, a deb.Architecture) error {
	return deb.NotYetImplemented()
}

func (x *Interactor) RemoveDistribution(d deb.Codename, a deb.Architecture) error {
	return deb.NotYetImplemented()
}

func (x *Interactor) ListDistributions() map[deb.Codename][]deb.Architecture {
	return nil
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

	_, ok := x.repo.List()[string(changes.Dist)]
	if ok == false {
		return res, fmt.Errorf("Unsupported distribution %s", changes.Dist)
	}

	return res, deb.NotYetImplemented()
}
