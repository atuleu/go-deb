package main

import (
	"bytes"
	"fmt"
	"io"

	deb ".."
)

type UserDistributionSupport interface {
	Add(d deb.Distribution, a deb.Architecture) error
	Supported() map[deb.Distribution][]deb.Architecture
	Remove(d deb.Distribution, a deb.Architecture) error
}

type DistributionInitResult struct {
	Message   string
	CreateLog Log
}

func (x *Interactor) AddDistributionSupport(d deb.Distribution, a deb.Architecture, output io.Writer) (*DistributionInitResult, error) {
	supported := false
	for _, aa := range x.builder.AvailableArchitectures(d) {
		if aa == a {
			supported = true
			break
		}
	}

	res := &DistributionInitResult{}

	if supported == false {
		var createOut bytes.Buffer
		var w io.Writer
		if output == nil {
			w = &createOut
		} else {
			w = io.MultiWriter(&createOut, output)
		}
		err := x.builder.InitDistribution(d, a, w)
		res.CreateLog = Log(createOut.String())
		if err != nil {
			res.Message = fmt.Sprintf("Could not initialize `%s' with architecture `%s'", d, a)
			return res, err
		} else {
			res.Message = fmt.Sprintf("Initialized `%s' with architecture `%s'", d, a)
		}
	} else {
		res.Message = fmt.Sprintf("Distribution `%s' already supports `%s'", d, a)
	}

	err := x.u.Add(d, a)
	if err != nil {
		return res, fmt.Errorf("Could not modify user settings : %s", err)
	}

	return res, nil
}

func (x *Interactor) RemoveDistributionSupport(d deb.Distribution, a deb.Architecture, deleteCache bool) error {
	err := x.u.Remove(d, a)
	if err != nil {
		return err
	}

	if deleteCache == false {
		return nil
	}

	return x.builder.RemoveDistribution(d, a)
}

func (x *Interactor) GetSupportedDistribution() map[deb.Distribution][]deb.Architecture {
	return x.u.Supported()
}

func (x *Interactor) UpdateDistribution(d deb.Distribution, a deb.Architecture) error {
	supported := false
	for _, aa := range x.builder.AvailableArchitectures(d) {
		if aa == a {
			supported = true
			break
		}
	}

	if supported == false {
		return fmt.Errorf("Distribution `%s' architecture `%s' is not supported, could not update it.", d, a)
	}

	return x.builder.UpdateDistribution(d, a)
}
