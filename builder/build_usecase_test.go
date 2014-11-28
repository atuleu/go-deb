package main

import (
	"fmt"
	"io"
	"testing"

	deb "../"
	. "gopkg.in/check.v1"
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

func (b *DebianBuilderStub) InitDistribution(d DistributionAndArch) error {
	if b.Err != nil {
		return b.Err
	}
	b.DistAndArch[d.Dist] = append(b.DistAndArch[d.Dist], d.Arch)
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

	p.Results[b.Changes.Identifier] = &b
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

type HistoryStub struct {
	hist []deb.SourcePackageRef
}

func (h *HistoryStub) Append(p deb.SourcePackageRef) {
	h.hist = append([]deb.SourcePackageRef{p}, h.hist...)
}
func (h *HistoryStub) Get() []deb.SourcePackageRef {
	return h.hist
}
func (h *HistoryStub) RemoveFront(p deb.SourcePackageRef) {
	oldHist := h.hist
	h.hist = nil
	for _, oldP := range oldHist {
		if oldP == p {
			continue
		}
		h.hist = append(h.hist, oldP)
	}
}

type BuildUseCaseSuite struct {
	x               Interactor
	builder         *DebianBuilderStub
	packageArchiver *PackageArchiverStub
	history         *HistoryStub
	dsc             deb.SourceControlFile
}

var _ = Suite(&BuildUseCaseSuite{})

func Test(t *testing.T) { TestingT(t) }

func (s *BuildUseCaseSuite) TestBuildDebianizedGit(c *C) {
	r, err := s.x.BuildDebianizedGit("", nil)
	c.Check(r, IsNil)
	c.Check(err, ErrorMatches, ".* is not yet implemented")
}

func (s *BuildUseCaseSuite) TestBuildAutobuildSource(c *C) {
	r, err := s.x.BuildAutobuildSource(AutobuildSourcePackage{}, nil)
	c.Check(r, IsNil)
	c.Check(err, ErrorMatches, ".* is not yet implemented")
}

func (s *BuildUseCaseSuite) SetUpTest(c *C) {
	s.dsc = deb.SourceControlFile{
		Identifier: deb.SourcePackageRef{
			Source: "foo-software",
			Ver: deb.Version{
				Epoch:           0,
				UpstreamVersion: "1.2.3",
				DebianRevision:  "1",
			},
		},
	}

	s.builder = &DebianBuilderStub{
		DistAndArch: map[deb.Distribution][]deb.Architecture{},
		Res: &BuildResult{
			Changes: &deb.ChangesFileRef{
				Identifier: s.dsc.Identifier,
				Suffix:     "multi",
			},
			BasePath: "/dev/null",
		},
	}

	s.packageArchiver = &PackageArchiverStub{
		Sources:         make(map[deb.SourcePackageRef]*ArchivedSource),
		Results:         make(map[deb.SourcePackageRef]*BuildResult),
		ForceTargetDist: "unstable",
	}

	s.builder.InitDistribution(DistributionAndArch{
		Dist: "unstable",
		Arch: deb.Amd64,
	})
	s.history = &HistoryStub{}
	s.x.h = s.history
	s.x.b = s.builder
	s.x.p = s.packageArchiver
}

func (s *BuildUseCaseSuite) TestWorkingWorkflow(c *C) {

	b, err := s.x.BuildPackage(s.dsc, nil)

	c.Check(err, IsNil)
	c.Check(b, DeepEquals, s.builder.Res)

	//calls check
	c.Check(s.builder.BuildCalled, Equals, true)
	c.Check(s.packageArchiver.ArchiveSourceCalled, Equals, true)
	c.Check(s.packageArchiver.ArchiveResultCalled, Equals, true)

	//other tests
	c.Check(*s.x.GetLastSuccesfullUserBuild(), DeepEquals, s.dsc.Identifier)
	res, err := s.x.GetBuildResult(s.dsc.Identifier)
	c.Assert(err, IsNil)
	c.Check(res, DeepEquals, s.builder.Res)
}

func (s *BuildUseCaseSuite) TestBuildCouldNotArchiveSource(c *C) {
	s.packageArchiver.SourceErr = fmt.Errorf("Failure")

	b, err := s.x.BuildPackage(s.dsc, nil)
	c.Check(b, IsNil)
	c.Check(err, ErrorMatches, "Could not archive source package `.*': Failure")
	c.Check(s.builder.BuildCalled, Equals, false)
	c.Check(s.packageArchiver.ArchiveSourceCalled, Equals, false)
	c.Check(s.packageArchiver.ArchiveResultCalled, Equals, false)
	c.Check(s.x.GetLastSuccesfullUserBuild(), IsNil)

}

func (s *BuildUseCaseSuite) TestBuildCouldNotBuildButArchive(c *C) {
	s.builder.Err = fmt.Errorf("Failure")

	b, err := s.x.BuildPackage(s.dsc, nil)

	c.Check(b, NotNil)
	c.Check(err, ErrorMatches, "Failure")
	c.Check(s.builder.BuildCalled, Equals, true)
	c.Check(s.packageArchiver.ArchiveSourceCalled, Equals, true)
	c.Check(s.packageArchiver.ArchiveResultCalled, Equals, true)

	c.Check(s.x.GetLastSuccesfullUserBuild(), IsNil)

}

func (s *BuildUseCaseSuite) TestBuildCouldBuildButNotArchive(c *C) {
	s.packageArchiver.BuildErr = fmt.Errorf("Failure")

	s.history.hist = []deb.SourcePackageRef{s.dsc.Identifier}
	b, err := s.x.BuildPackage(s.dsc, nil)
	c.Check(b, IsNil)
	c.Check(err, ErrorMatches, "Failed to archive build result of `.*': Failure")
	c.Check(s.builder.BuildCalled, Equals, true)
	c.Check(s.packageArchiver.ArchiveSourceCalled, Equals, true)
	c.Check(s.packageArchiver.ArchiveResultCalled, Equals, false)

	c.Check(s.x.GetLastSuccesfullUserBuild(), IsNil)
}

func (s *BuildUseCaseSuite) TestBuildUnsupportedDistribution(c *C) {
	s.packageArchiver.ForceTargetDist = "sid"

	b, err := s.x.BuildPackage(s.dsc, nil)

	c.Check(b, IsNil)
	c.Check(err, ErrorMatches, "Target distribution `.*' of source package `.*' is not supported")
	c.Check(s.builder.BuildCalled, Equals, false)
	c.Check(s.packageArchiver.ArchiveSourceCalled, Equals, true)
	c.Check(s.packageArchiver.ArchiveResultCalled, Equals, false)

	c.Check(s.x.GetLastSuccesfullUserBuild(), IsNil)
}
