package main

import "fmt"

type Interactor struct {
	archiver        PackageArchiver
	localRepository AptRepository
	builder         DebianBuilder
	history         History
	u               UserDistributionSupport
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

	return res, nil
}
