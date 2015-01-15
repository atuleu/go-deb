package main

import (
	"bytes"
	"fmt"
	"io"

	deb ".."
)

type ArchitectureList []deb.Architecture

func (l ArchitectureList) Len() int {
	return len(l)
}

func (l ArchitectureList) Less(i, j int) bool {
	return l[i] < l[j]
}

func (l ArchitectureList) Swap(i, j int) {
	tmp := l[i]
	l[i] = l[j]
	l[j] = tmp
}

type UserDistSupportConfig interface {
	Add(d deb.Codename, a deb.Architecture) error
	Supported() map[deb.Codename]ArchitectureList
	Remove(d deb.Codename, a deb.Architecture) error
}

type DistributionInitResult struct {
	Message   string
	CreateLog Log
}

func (x *Interactor) AddDistributionSupport(d deb.Codename, a deb.Architecture, output io.Writer) (*DistributionInitResult, error) {
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
			res.Message = fmt.Sprintf("Builder could not initialize distribution %s-%s", d, a)
			return res, err
		} else {
			res.Message = fmt.Sprintf("Builder initialized %s-%s", d, a)
		}
	}

	err := x.userDistConfig.Add(d, a)
	if err != nil {
		return res, fmt.Errorf("Could not modify user settings : %s", err)
	}

	err = x.localRepository.AddDistribution(d, a)
	if err != nil {
		return nil, err
	}

	var sep string
	if len(res.Message) > 0 {
		sep = "\n"
	}
	res.Message = res.Message + sep + fmt.Sprintf("Enabled user distribution support for %s-%s", d, a)

	return res, nil
}

func (x *Interactor) RemoveDistributionSupport(d deb.Codename, a deb.Architecture, removeBuilder bool) error {
	err := x.userDistConfig.Remove(d, a)
	if err != nil {
		return err
	}

	err = x.localRepository.RemoveDistribution(d, a)
	if err != nil {
		return err
	}

	if removeBuilder == false {
		return nil
	}

	return x.builder.RemoveDistribution(d, a)
}

type DistributionSupportReport map[deb.Codename]map[deb.Architecture]bool

func (x *Interactor) GetSupportedDistribution() (DistributionSupportReport, error) {
	res := make(DistributionSupportReport)

	for _, d := range x.builder.AvailableDistributions() {
		res[d] = make(map[deb.Architecture]bool)
		for _, a := range x.builder.AvailableArchitectures(d) {
			res[d][a] = false
		}
	}
	for d, archs := range x.userDistConfig.Supported() {
		for _, a := range archs {
			builderArchs, okDist := res[d]
			if okDist == false {
				return nil, fmt.Errorf("System consistency error: user list distributions %s:%v, but builder does not support %s", d, archs, d)
			}
			_, okArch := builderArchs[a]
			if okArch == false {
				return nil, fmt.Errorf("System consistency error: user list distributions %s:%v, but builder does not support %s for %s", d, archs, a, d)
			}
			res[d][a] = true
		}
	}

	return res, nil
}

func (x *Interactor) UpdateDistribution(d deb.Codename, a deb.Architecture, output io.Writer) error {
	supported := false
	for _, aa := range x.builder.AvailableArchitectures(d) {
		if aa == a {
			supported = true
			break
		}
	}

	if supported == false {
		return fmt.Errorf("Distribution %s-%s is not supported by builder, could not update it.", d, a)
	}

	return x.builder.UpdateDistribution(d, a, output)
}
