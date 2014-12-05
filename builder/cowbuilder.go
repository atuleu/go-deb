package main

import (
	"io"
	"path"
	"runtime"

	deb ".."
	"github.com/nightlyone/lockfile"
)

type Cowbuilder struct {
	basepath string
	lock     lockfile.Lockfile

	semaphore chan bool
}

func NewCowbuilder(basepath string) (*Cowbuilder, error) {
	res := &Cowbuilder{
		basepath: basepath,
	}
	var err error

	res.lock, err = lockfile.New(path.Join(basepath, "global.lock"))
	if err != nil {
		return nil, err
	}

	err = res.lock.TryLock()
	if err != nil {
		return nil, err
	}
	runtime.SetFinalizer(res.lock, res.lock.Unlock())

	res.semaphore = make(chan bool, 1)
	res.release()

	return res, nil
}

func (b *Cowbuilder) acquire() {
	_ = <-b.semaphore
}

func (b *Cowbuilder) release() {
	b.semaphore <- true
}

func (b *Cowbuilder) BuildPackage(p deb.SourceControlFile, output io.Writer) (*BuildResult, error) {
	b.acquire()
	defer b.release()

	return nil, deb.NotYetImplemented()
}

func (b *Cowbuilder) InitDistribution(d DistributionAndArch, output io.Writer) error {
	b.acquire()
	defer b.release()

	return deb.NotYetImplemented()
}

func (b *Cowbuilder) RemoveDistribution(DistributionAndArch) error {
	b.acquire()
	defer b.release()

	return deb.NotYetImplemented()
}

func (b *Cowbuilder) UpdateDistribution(DistributionAndArch) error {
	b.acquire()
	defer b.release()

	return deb.NotYetImplemented()
}

func (b *Cowbuilder) AvailableDistributions() []deb.Distribution {
	b.acquire()
	defer b.release()

	return nil
}

func (b *Cowbuilder) AvailableArchitectures(d deb.Distribution) []deb.Architecture {
	b.acquire()
	defer b.release()

	return nil
}
