package main

import (
	"fmt"
	"io"

	deb ".."
)

type DebianBuilderStub struct {
	Err         error
	Res         *BuildResult
	BuildCalled bool
	DistAndArch map[deb.Distribution][]deb.Architecture
}

func (b *DebianBuilderStub) BuildPackage(p deb.SourceControlFile, out io.Writer) (*BuildResult, error) {
	b.BuildCalled = true
	return b.Res, b.Err
}

func (b *DebianBuilderStub) InitDistribution(d DistributionAndArch, out io.Writer) error {
	if b.Err != nil {
		return b.Err
	}
	b.DistAndArch[d.Dist] = append(b.DistAndArch[d.Dist], d.Arch)
	return nil
}

func (b *DebianBuilderStub) RemoveDistribution(d DistributionAndArch) error {
	if b.Err != nil {
		return b.Err
	}
	archs, ok := b.DistAndArch[d.Dist]
	if ok == false {
		return fmt.Errorf("Distribution `%s' is not supported", d.Dist)
	}
	newArch := make([]deb.Architecture, 0, cap(archs))
	found := false
	for _, a := range archs {
		if a == d.Arch {
			found = true
			continue
		}
		newArch = append(newArch, a)
	}
	if found == false {
		return fmt.Errorf("Distribution `%s' doas not support architecture `%s'", d.Dist, d.Arch)
	}
	b.DistAndArch[d.Dist] = newArch
	return nil
}

func (b *DebianBuilderStub) UpdateDistribution(d DistributionAndArch) error {
	archs, ok := b.DistAndArch[d.Dist]
	if ok == false {
		return fmt.Errorf("Distribution %s is not supported", d.Dist)
	}

	archSupported := false
	for _, a := range archs {
		if a == d.Arch {
			archSupported = true
			break
		}
	}

	if archSupported == false {
		return fmt.Errorf("Architecture %s of %s is not supported", d.Arch, d.Dist)
	}

	return b.Err

}

func (b *DebianBuilderStub) AvailableDistributions() []deb.Distribution {
	res := []deb.Distribution{}

	for d, _ := range b.DistAndArch {
		res = append(res, d)
	}

	return res
}

func (b *DebianBuilderStub) AvailableArchitectures(d deb.Distribution) []deb.Architecture {
	return b.DistAndArch[d]
}

func (b *DebianBuilderStub) CurrentBuild() *InBuildResult {
	return nil
}
