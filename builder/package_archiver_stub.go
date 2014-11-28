package main

import (
	"fmt"

	deb ".."
)

type PackageArchiverStub struct {
	ArchiveSourceCalled bool
	ArchiveResultCalled bool
	SourceErr           error
	BuildErr            error
	ForceTargetDist     deb.Distribution
	Sources             map[deb.SourcePackageRef]*ArchivedSource
	Results             map[deb.SourcePackageRef]*BuildResult
}

func (pa *PackageArchiverStub) ArchiveSource(p deb.SourceControlFile) (*ArchivedSource, error) {
	if pa.SourceErr != nil {
		return nil, pa.SourceErr
	}
	res := &ArchivedSource{
		Changes: &deb.ChangesFileRef{
			Identifier: p.Identifier,
			Suffix:     "sources",
		},
		Dsc:        p,
		TargetDist: pa.ForceTargetDist,
		BasePath:   "/dev/null",
	}
	pa.ArchiveSourceCalled = true
	pa.Sources[p.Identifier] = res
	return res, nil
}

func (p *PackageArchiverStub) ArchiveBuildResult(b BuildResult) (*BuildResult, error) {
	if p.BuildErr != nil {
		return nil, p.BuildErr
	}

	p.Results[b.Changes.Ref.Identifier] = &b
	p.ArchiveResultCalled = true
	return &b, nil
}

func (pa *PackageArchiverStub) GetArchivedSource(p deb.SourcePackageRef) (*ArchivedSource, error) {
	a, ok := pa.Sources[p]
	if ok == false {
		return nil, fmt.Errorf("Could not found %s sources", p)
	}
	return a, nil
}

func (pa *PackageArchiverStub) GetBuildResult(p deb.SourcePackageRef) (*BuildResult, error) {
	b, ok := pa.Results[p]
	if ok == false {
		return nil, fmt.Errorf("Could not found %s build results", p)
	}
	return b, nil
}
