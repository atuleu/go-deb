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

func (b *DebianBuilderStub) BuildPackage(args BuildArguments, out io.Writer) (*BuildResult, error) {
	b.BuildCalled = true
	return b.Res, b.Err
}

func (b *DebianBuilderStub) InitDistribution(d deb.Distribution, a deb.Architecture, out io.Writer) error {
	if b.Err != nil {
		return b.Err
	}
	b.DistAndArch[d] = append(b.DistAndArch[d], a)
	return nil
}

func (b *DebianBuilderStub) RemoveDistribution(d deb.Distribution, a deb.Architecture) error {
	if b.Err != nil {
		return b.Err
	}
	archs, ok := b.DistAndArch[d]
	if ok == false {
		return fmt.Errorf("Distribution `%s' is not supported", d)
	}
	newArch := make([]deb.Architecture, 0, cap(archs))
	found := false
	for _, aa := range archs {
		if aa == a {
			found = true
			continue
		}
		newArch = append(newArch, aa)
	}
	if found == false {
		return fmt.Errorf("Distribution `%s' doas not support architecture `%s'", d, a)
	}
	if len(newArch) > 0 {
		b.DistAndArch[d] = newArch
	} else {
		delete(b.DistAndArch, d)
	}
	return nil
}

func (b *DebianBuilderStub) UpdateDistribution(d deb.Distribution, a deb.Architecture, output io.Writer) error {
	archs, ok := b.DistAndArch[d]
	if ok == false {
		return fmt.Errorf("Distribution %s is not supported", d)
	}

	archSupported := false
	for _, aa := range archs {
		if aa == a {
			archSupported = true
			break
		}
	}

	if archSupported == false {
		return fmt.Errorf("Architecture %s of %s is not supported", d, a)
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

func (b *DebianBuilderStub) AvailableArchitectures(d deb.Distribution) ArchitectureList {
	return b.DistAndArch[d]
}

func (b *DebianBuilderStub) CurrentBuild() *InBuildResult {
	return nil
}
