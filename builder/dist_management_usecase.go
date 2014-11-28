package main

import (
	"bytes"
	"fmt"
	"io"
)

type UserDistributionSupport interface {
	Add(d DistributionAndArch) error
	Supported() []DistributionAndArch
	Remove(d DistributionAndArch) error
}

type DistributionInitResult struct {
	Message   string
	CreateLog Log
}

func (x *Interactor) AddDistributionSupport(d DistributionAndArch, output io.Writer) (*DistributionInitResult, error) {
	supported := false
	for _, a := range x.b.AvailableArchitectures(d.Dist) {
		if a == d.Arch {
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
		err := x.b.InitDistribution(d, w)
		res.CreateLog = Log(createOut.String())
		if err != nil {
			res.Message = fmt.Sprintf("Could not initialize `%s' with architecture `%s'", d.Dist, d.Arch)
			return res, err
		} else {
			res.Message = fmt.Sprintf("Initialized `%s' with architecture `%s'", d.Dist, d.Arch)
		}
	} else {
		res.Message = fmt.Sprintf("Distribution `%s' already supports `%s'", d.Dist, d.Arch)
	}

	err := x.u.Add(d)
	if err != nil {
		return res, fmt.Errorf("Could not modify user settings : %s", err)
	}

	return res, nil
}

func (x *Interactor) RemoveDistributionSupport(d DistributionAndArch, deleteCache bool) error {
	err := x.u.Remove(d)
	if err != nil {
		return err
	}

	if deleteCache == false {
		return nil
	}

	return x.b.RemoveDistribution(d)
}

func (x *Interactor) GetSupportedDistribution() []DistributionAndArch {
	return x.u.Supported()
}

func (x *Interactor) UpdateDistribution(d DistributionAndArch) error {
	supported := false
	for _, a := range x.b.AvailableArchitectures(d.Dist) {
		if a == d.Arch {
			supported = true
			break
		}
	}

	if supported == false {
		return fmt.Errorf("Distribution `%s' architecture `%s' is not supported, could not update it.", d.Dist, d.Arch)
	}

	return x.b.UpdateDistribution(d)
}
