package main

import (
	"bytes"

	deb ".."
	. "gopkg.in/check.v1"
)

type AptDepUseCasesSuite struct {
	x       Interactor
	aptDeps *AptDepsManagerStub
}

var _ = Suite(&AptDepUseCasesSuite{})

type AptDepsManagerStub struct {
	data map[AptRepositoryID]*AptRepositoryAccess
}

func (s *AptDepsManagerStub) Store(a *AptRepositoryAccess) error {
	s.data[a.ID] = a
	return nil
}

func (s *AptDepsManagerStub) Remove(id AptRepositoryID) error {
	delete(s.data, id)
	return nil
}

func (s *AptDepsManagerStub) List() map[AptRepositoryID]*AptRepositoryAccess {
	return s.data
}

var ponyoPgpArmoredKey = `-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: GnuPG v1

mQINBFS9Dt4BEACcSMRiAIfqrm5kAHk2KT/vADtx5fiL1b65jjCS27CyQ99Q7mx2
RcDRnr4XJhaBDEAAp9G0mqer2sUiodedShK4hqx9ht2Zztr9LN1BztHKZokVQZhr
wKzTJqY7KNLdVreXisoziVlsftaUuKUkqGHyXr2Fuem1VhfokU6RW1yLG4i9P70J
JeMpGeBIEgJu7mij3HD/6VY3iBArx7BSTs9Ql1OzE8lUNW8INkqlB4OHbHkLCPsR
phW6OpDZBy0wsfZIFVRtEB1E0ZmUalWjfLkGs+MVrqVlo4n6woC4uq1vJwB31uCY
7CDGIAAqjdwlzOlJJiaUWWiiCRR0rNXrEOBWLW8YINUJ/ej6ihi1yRxD/KGR+6Qt
D5/lZTbZIbPNVLserwXBqX2I8M9yRnyFnK2q8i2Qo7w5rqA3JZS/Gy9wtN16akon
Cu/gMqrhv/WFjwxTwnr6lZAbGDETqbom8Fmpdrw1VlzapL/X+2aVaMkH1B2mOb0Y
tNlHGHmsZRLwh0WCnA3tKiUwI6GRzy0MaXENwokwJCV8hABLDVM0+JTBMEtsdNLC
ICHmcKaoUQKGUo7Srxi+TiO16E2j2U+qHA5QAmluhHBOjzThJ8P8NrokS4UgV+Qv
zvZEgyjW1KtelN0TUwbdeoVn7vZd6s6EtQaQ5MErvs6SkczF2/MHrSTZ1wARAQAB
tDlCaW9Sb2IgTWFpbnRhaW5lcnMgKFVwbG9hZGVyIEtleSkgPHN1cHBvcnRAcG9u
eW8uZXBmbC5jaD6JAjgEEwECACIFAlS9Dt4CGy8GCwkIBwMCBhUIAgkKCwQWAgMB
Ah4BAheAAAoJEKrce/+cRSHtiE0P/2Fqg0CibDXr1Kr37VavGzb+Bv0tG4PVAh2R
tvfsz8yO2zrYhQMe4jTeIrUgw3XHrZDM/MCOyvbmZhC9/Hv1yYgXTRq7dxrjZI90
WrVzgb33idapFBkotBLQ7Lk/ceU6mwGoXWzCl4+bVCsjHaztgKl+TGv/OO0dq5Em
mGVaaCbc96NeChO3aj1d/z/OfVcPlqO+XZ7dGDBkmg9qAhn/nSrv4ei2+Zsf5lxR
x+Dy+fag4Mz/oUifk5rRwWD0PvkynCKk/YCSY/7h3ccyCo9oRhAjnX350tzdRfL1
4tldSIhurthZIUZ0OavIU6EMB97R5qZVMIB+Th/PIm8kGSc3LYZq5pUfRxw6jJ5L
yjx2hFzinVFNXQkbBE8drdcdPNBKvqvUXjZZ4cXmTC5l7aEwaLADC+yuhjscIJEK
xY1R08paKkVhEyc8rV6j5TQU4bDZbQJ4qgs/uwzEV/EuEOcinnsuEZBxPZdG99QQ
qCuktKCai6FUBbssWbY0INComRzSNn7VBTcD7VJkO8vymTyMaNf8nqLTgIhysr6A
1YpV8gRRCQ4+4Rl7jS1PzpamxzytViauudFFFgxrymhL8RuKkbvbJKdPbj7zLeOH
jBN5RdlBqmKyQFjQfUqJXvowUq07k2z5kW7Jr9IwBRqLZAgdt0/62QOwBK6UQBEe
dob6pYd8
=N8fv
-----END PGP PUBLIC KEY BLOCK-----
`

func (s *AptDepUseCasesSuite) SetUpTest(c *C) {
	s.aptDeps = &AptDepsManagerStub{
		data: map[AptRepositoryID]*AptRepositoryAccess{
			"ppa:foo/bar": &AptRepositoryAccess{
				ID:      "ppa:foo/bar",
				Address: "http://ppa.launchpad.net/foo/bar/ubuntu",
			},
			"http://foobar.com": &AptRepositoryAccess{
				ID:               "http://foobar.com",
				Address:          "http://foobar.com",
				ArmoredPublicKey: []byte(ponyoPgpArmoredKey),
			},
		},
	}
	s.x.aptDeps = s.aptDeps
}

func (s *AptDepUseCasesSuite) TestPPACreation(c *C) {

	id, err := s.x.CreatePPADependency("foo:bar/baz")
	c.Check(len(id), Equals, 0)
	c.Check(err, ErrorMatches, "Invalid PPA address .*")

	id, err = s.x.CreatePPADependency("ppa:foo/bar")
	c.Check(len(id), Equals, 0)
	c.Check(err, ErrorMatches, "Repository .* already exists")

	id, err = s.x.CreatePPADependency("ppa:tuleu/ppa")
	c.Assert(id, Equals, AptRepositoryID("ppa:tuleu/ppa"))
	c.Assert(err, IsNil)
}

func (s *AptDepUseCasesSuite) TestRemoteCreation(c *C) {
	id, err := s.x.CreateRemoteDependency("http://foo", nil)
	c.Check(len(id), Equals, 0)
	c.Check(err, ErrorMatches, "Could not create new remote repository without a PGP public key")

	id, err = s.x.CreateRemoteDependency("http://foo", bytes.NewBuffer([]byte(" ")))
	c.Check(len(id), Equals, 0)
	c.Check(err, ErrorMatches, "Invalid key file, expected a single key but got 0")

	id, err = s.x.CreateRemoteDependency("http://foobar.com", bytes.NewBuffer([]byte(" ")))
	c.Check(len(id), Equals, 0)
	c.Check(err, ErrorMatches, "Repository .* already exists")

	id, err = s.x.CreateRemoteDependency("http://foo.com",
		bytes.NewBuffer([]byte(ponyoPgpArmoredKey)))
	c.Check(id, Equals, AptRepositoryID("http://foo.com"))
	c.Check(err, IsNil)

}

func (s *AptDepUseCasesSuite) TestRemove(c *C) {

	err := s.x.RemoveDependency("ppa:foo/bar")
	c.Check(err, IsNil)
	c.Check(s.aptDeps.data[AptRepositoryID("ppa:foo/bar")], IsNil)
	err = s.x.RemoveDependency("ppa:foo/bar")
	c.Check(err, IsNil)

}

func (s *AptDepUseCasesSuite) TestPPAEdition(c *C) {
	err := s.x.EditRepository("ppa:foo/bar",
		map[deb.Codename][]deb.Component{
			deb.Trusty:  []deb.Component{"main"},
			deb.Precise: nil,
		}, nil)
	c.Check(err, IsNil)
	c.Check(len(s.aptDeps.List()["ppa:foo/bar"].Components), Equals, 2)

	err = s.x.EditRepository("ppa:foo/bar",
		map[deb.Codename][]deb.Component{
			deb.Precise: []deb.Component{"testing"},
		}, nil)
	c.Check(err, ErrorMatches, "PPA repository can only list main, but .* asked")

	err = s.x.EditRepository("ppa:foo/bar",
		nil,
		map[deb.Codename][]deb.Component{
			deb.Trusty:  []deb.Component{"main"},
			deb.Precise: []deb.Component{"main"},
		})

	c.Check(err, IsNil)
	c.Check(s.aptDeps.List()["ppa:foo/bar"], IsNil)

}

func (s *AptDepUseCasesSuite) TestRemoteEdition(c *C) {
	err := s.x.EditRepository("https:foo.com", nil, nil)
	c.Check(err, ErrorMatches, "Unknown repository .*")

	err = s.x.EditRepository("http://foobar.com",
		map[deb.Codename][]deb.Component{
			deb.Trusty:  []deb.Component{"main", "testing", "multiverse"},
			deb.Precise: nil,
		}, nil)
	c.Check(err, IsNil)
	access := s.aptDeps.List()["http://foobar.com"]
	c.Assert(access, NotNil)
	c.Check(len(access.Components), Equals, 1)
	// could not DeepEqual because order could be random
	c.Check(len(access.Components[deb.Trusty]), Equals, 3)
}

func (s *AptDepUseCasesSuite) TestList(c *C) {
	list := s.x.ListDependencies()
	c.Assert(len(list), Equals, 2)
}
