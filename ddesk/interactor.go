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
	auth            DebfileAuthentifier
}

func NewInteractor(o *Options) (*Interactor, error) {

	if o.BuilderType != "client" {
		return nil, fmt.Errorf("Only client builder are supported, it means that you have to run a `sudo ddesk serve-builder`")
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

	res.localRepository, err = NewReprepro(path.Join(xdg.Data.Home(), "go-deb.ddesk/local_reprepro"))
	if err != nil {
		return nil, err
	}

	res.history, err = NewXdgHistory()
	if err != nil {
		return nil, err
	}

	res.auth, err = NewAuthentifier()
	if err != nil {
		return nil, err
	}

	res.archiver, err = NewXdgArchiver(res.auth)
	if err != nil {
		return nil, err
	}

	return res, nil
}
