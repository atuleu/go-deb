package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"

	deb ".."
)

func (x *Interactor) CreatePPADependency(address string) (AptRepositoryID, error) {
	access, err := NewPPArepositoryAccess(address)
	if err != nil {
		return "", err
	}

	_, ok := x.aptDeps.List()[access.ID]
	if ok == true {
		return "", fmt.Errorf("Repository %s already exists", access)
	}

	return access.ID, x.aptDeps.Store(access)
}

func (x *Interactor) CreateRemoteDependency(address string, keyReader io.Reader) (AptRepositoryID, error) {
	access := &AptRepositoryAccess{
		Address: address,
		ID:      AptRepositoryID(address),
	}

	if keyReader == nil {
		return "", fmt.Errorf("Could not create new remote repository without a PGP public key")
	}

	_, ok := x.aptDeps.List()[access.ID]
	if ok == true {
		return "", fmt.Errorf("Repository %s already exists", access)
	}

	keys, err := openpgp.ReadArmoredKeyRing(keyReader)
	if err != nil && err.Error() == "no armor data found" {
		keys, err = openpgp.ReadKeyRing(keyReader)
	}
	if err != nil {
		return "", err
	}
	if len(keys) != 1 {
		return "", fmt.Errorf("Invalid key file, expected a single key but got %d", len(keys))
	}

	var armoredData bytes.Buffer
	w, err := armor.Encode(&armoredData, "PGP PUBLIC KEY BLOCK", nil)
	if err != nil {
		return "", err
	}
	defer w.Close()

	err = keys[0].Serialize(w)
	if err != nil {
		return "", err
	}
	w.Close()
	access.ArmoredPublicKey = armoredData.Bytes()

	return access.ID, x.aptDeps.Store(access)
}

func (x *Interactor) EditRepository(id AptRepositoryID, toAdd, toRemove map[deb.Codename][]deb.Component) error {
	access, ok := x.aptDeps.List()[id]
	if ok == false {
		return fmt.Errorf("Unknown repository %s", id)
	}

	if access.Components == nil {
		access.Components = make(map[deb.Codename][]deb.Component)
	}

	isPPa := strings.HasPrefix(string(id), "ppa:")

	//add new comps
	for d, toAddComps := range toAdd {
		compSet := make(map[deb.Component]bool)
		for _, c := range access.Components[d] {
			compSet[c] = true
		}

		if isPPa == true {
			if len(toAddComps) > 1 ||
				(len(toAddComps) == 1 && toAddComps[0] != "main") {
				return fmt.Errorf("PPA repository can only list main, but %v asked", toAddComps)
			}
			toAddComps = []deb.Component{"main"}
		}

		for _, c := range toAddComps {
			compSet[c] = true
		}
		access.Components[d] = make([]deb.Component, 0, len(compSet))

		if len(compSet) == 0 {
			delete(access.Components, d)
			continue
		}

		for c, _ := range compSet {
			access.Components[d] = append(access.Components[d], c)
		}
	}

	for d, toRemoveComps := range toRemove {
		compSet := make(map[deb.Component]bool)
		for _, c := range access.Components[d] {
			compSet[c] = true
		}
		for _, c := range toRemoveComps {
			delete(compSet, c)
		}
		if len(compSet) == 0 {
			delete(access.Components, d)
			continue
		}
		access.Components[d] = make([]deb.Component, 0, len(compSet))
		for c, _ := range compSet {
			access.Components[d] = append(access.Components[d], c)
		}
	}

	if len(access.Components) == 0 {
		return x.RemoveDependency(id)
	}

	return x.aptDeps.Store(access)
}

func (x *Interactor) RemoveDependency(id AptRepositoryID) error {
	return x.aptDeps.Remove(id)
}

type DependencyItem struct {
	Components map[deb.Codename][]deb.Component
	KeyID      string
}

func (x *Interactor) ListDependencies() map[AptRepositoryID]DependencyItem {
	res := make(map[AptRepositoryID]DependencyItem)
	for id, access := range x.aptDeps.List() {
		item := DependencyItem{
			Components: access.Components,
		}
		item.KeyID = "none"
		if len(access.ArmoredPublicKey) != 0 {
			item.KeyID = "error"
			keys, err := openpgp.ReadArmoredKeyRing(bytes.NewBuffer(access.ArmoredPublicKey))
			if err != nil {
				break
			}

			if len(keys) != 1 {
				break
			}
			item.KeyID = keys[0].PrimaryKey.KeyIdShortString()
		}
		res[id] = item
	}
	return res
}
