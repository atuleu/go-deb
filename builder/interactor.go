package main

import (
	"fmt"
	"path"

	"launchpad.net/go-xdg"
)

type Interactor struct {
	archiver        PackageArchiver
	localRepository AptRepository
	builder         DebianBuilder
	history         History
	userDistConfig  UserDistSupportConfig
}

func NewInteractor(o *Options) (*Interactor, error) {

	if o.BuilderType != "client" {
		return nil, fmt.Errorf("Only client builder are supported, it means that you ahve to run a `sudo builder serve-builder`")
	}

	res := &Interactor{}
	var err error
	res.builder, err = NewClientBuilder("unix", o.BuilderSocket)
	if err != nil {
		return nil, err
	}

	res.userDistConfig, err = NewXdgUserDistConfig()
	if err != nil {
		return nil, err
	}

	res.localRepository, err = NewReprepro(path.Join(xdg.Data.Home(), "go-deb.builder/local_reprepro"))
	if err != nil {
		return nil, err
	}

	res.history, err = NewXdgHistory()
	if err != nil {
		return nil, err
	}

	auth, err := NewAuthentifier()
	if err != nil {
		return nil, err
	}

	res.archiver, err = NewXdgArchiver(auth)
	if err != nil {
		return nil, err
	}

	return res, nil
}
