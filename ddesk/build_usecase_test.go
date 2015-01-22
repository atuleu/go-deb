package main

import (
	"fmt"
	"os"
	"testing"

	deb "../"
	. "gopkg.in/check.v1"
)

type BuildUseCaseSuite struct {
	x               Interactor
	aptDeps         *AptDepsManagerStub
	builder         *DebianBuilderStub
	packageArchiver *PackageArchiverStub
	localApt        *AptRepositoryStub
	history         *HistoryStub
	dsc             deb.SourceControlFile
	distConfig      *UserDistSupportConfigStub
}

type AptDepsManagerStub struct{}

func (a *AptDepsManagerStub) Store(*AptRepositoryAccess) error {
	return nil
}

func (a *AptDepsManagerStub) Remove(AptRepositoryID) error {
	return nil
}

func (a *AptDepsManagerStub) List() map[AptRepositoryID]*AptRepositoryAccess {
	return nil
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
	dscFile, err := os.Create(s.dsc.Filename())
	c.Assert(err, IsNil)
	dscFile.Close()

	s.builder = &DebianBuilderStub{
		DistAndArch: map[deb.Codename][]deb.Architecture{},
		Res: &BuildResult{
			Changes: &deb.ChangesFile{
				Ref: deb.ChangesFileRef{
					Identifier: s.dsc.Identifier,
					Suffix:     "multi",
				},
			},
			BasePath: "/dev/null",
		},
	}

	s.packageArchiver = &PackageArchiverStub{
		Sources:         make(map[deb.SourcePackageRef]*ArchivedSource),
		Results:         make(map[deb.SourcePackageRef]*BuildResult),
		ForceTargetDist: "unstable",
	}

	s.builder.InitDistribution("unstable", deb.Amd64, nil)

	s.localApt = &AptRepositoryStub{}
	s.localApt.AddDistribution("unstable", deb.Amd64)
	s.history = &HistoryStub{}
	s.distConfig = &UserDistSupportConfigStub{
		supported: map[deb.Codename]map[deb.Architecture]bool{
			"unstable": map[deb.Architecture]bool{deb.Amd64: true},
		},
	}
	s.aptDeps = &AptDepsManagerStub{}

	s.x.history = s.history
	s.x.builder = s.builder
	s.x.archiver = s.packageArchiver
	s.x.localRepository = s.localApt
	s.x.userDistConfig = s.distConfig
	s.x.aptDeps = s.aptDeps
}

func (s *BuildUseCaseSuite) TearDownTest(c *C) {
	err := os.Remove(s.dsc.Filename())
	c.Assert(err, IsNil)
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

func (s *BuildUseCaseSuite) TestBuildConsistencyError(c *C) {
	err := s.distConfig.Add("unstable", deb.I386)
	c.Assert(err, IsNil)

	b, err := s.x.BuildPackage(s.dsc, nil)

	c.Check(b, IsNil)
	c.Check(err, ErrorMatches, "System consistency error: builder does not support unstable-i386")
	c.Check(s.builder.BuildCalled, Equals, false)
	c.Check(s.packageArchiver.ArchiveSourceCalled, Equals, true)
	c.Check(s.packageArchiver.ArchiveResultCalled, Equals, false)

	c.Check(s.x.GetLastSuccesfullUserBuild(), IsNil)
}
