package main

import (
	"fmt"
	"io"
	"net/mail"
	"os"
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

func (x *Interactor) AddDistribution(d deb.Codename, archs []deb.Architecture, comps []deb.Component) error {
	return x.repo.Add(d, archs, comps)
}

func (x *Interactor) RemoveDistribution(d deb.Codename, archs []deb.Architecture, comps []deb.Component) error {
	return x.repo.Remove(d, archs, comps)
}

type DistributionSupport struct {
	Codename      deb.Codename
	Architectures []deb.Architecture
	Components    []deb.Component
}

func (x *Interactor) ListDistributions() []DistributionSupport {
	res := make([]DistributionSupport, 0, len(x.repo.List()))
	for codename, def := range x.repo.List() {
		res = append(res, DistributionSupport{
			Codename:      codename,
			Architectures: def.Architectures,
			Components:    def.Components,
		})
	}
	return res
}

type IncludeResult struct {
	SendTo        *mail.Address
	ShouldReport  bool
	FilesToRemove []*QueueFileReference
}

func (x *Interactor) ProcessChangesFile(ref *QueueFileReference, out io.Writer) (*IncludeResult, error) {

	res := &IncludeResult{
		SendTo:        nil,
		FilesToRemove: make([]*QueueFileReference, 0, 3),
		ShouldReport:  false,
	}

	if strings.HasSuffix(ref.Name, ".changes") == false {
		return res, fmt.Errorf("Invalid filename %s", ref.Name)
	}

	f, err := os.Open(ref.Path())
	if err != nil {
		return res, err
	}

	r, authErr := x.keyManager.CheckAndRemoveClearsigned(f)
	if authErr != nil {
		if r != nil {
			authErr = fmt.Errorf("Unauthorized .changes upload: %s", authErr)
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
	for _, fileToDelete := range changes.Md5Files {
		res.FilesToRemove = append(res.FilesToRemove, &QueueFileReference{
			Name:      fileToDelete.Name,
			dir:       ref.dir,
			Component: ref.Component,
		})
	}

	if authErr != nil {
		return res, authErr
	}

	return res, deb.NotYetImplemented()
}
