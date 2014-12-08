package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"

	deb ".."
	"github.com/nightlyone/lockfile"
)

type Cowbuilder struct {
	basepath  string
	imagepath string
	hookpath  string
	confpath  string

	lock lockfile.Lockfile

	supported []deb.Architecture

	semaphore chan bool
}

func NewCowbuilder(basepath string) (*Cowbuilder, error) {
	res := &Cowbuilder{
		basepath: basepath,
	}
	err := os.MkdirAll(basepath, 0755)
	if err != nil {
		return nil, err
	}

	res.lock, err = lockfile.New(path.Join(basepath, "global.lock"))
	if err != nil {
		return nil, err
	}

	err = res.lock.TryLock()
	if err != nil {
		return nil, err
	}
	runtime.SetFinalizer(res, res.lock.Unlock())

	res.semaphore = make(chan bool, 1)
	res.release()

	res.imagepath = path.Join(res.basepath, "images")
	res.hookpath = path.Join(res.basepath, "hooks")
	res.confpath = path.Join(res.basepath, ".pbuilderrc")

	//check path
	if _, err := os.Stat(res.confpath); err != nil {
		if os.IsNotExist(err) == false {
			return nil, fmt.Errorf("Could not check existence of %s:  %s", res.confpath, err)
		}
		_, err := os.Create(res.confpath)
		if err != nil {
			return nil, fmt.Errorf("Could not create %s: %s", res.confpath, err)
		}
	}

	err = os.MkdirAll(res.imagepath, 0755)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(res.hookpath, 0755)
	if err != nil {
		return nil, err
	}

	res.getSupportedArchitectures()

	return res, nil
}

func (b *Cowbuilder) BuildPackage(a BuildArguments, output io.Writer) (*BuildResult, error) {
	b.acquire()
	defer b.release()

	return nil, deb.NotYetImplemented()
}

func (b *Cowbuilder) InitDistribution(d deb.Distribution, a deb.Architecture, output io.Writer) error {
	b.acquire()
	defer b.release()

	imagePath, err := b.supportedDistributionPath(d, a)
	if err == nil {
		return fmt.Errorf("Distribution %s architecture %s is already supported", d, a)
	}

	supported := false
	for _, aa := range b.supported {
		if aa == a {
			supported = true
			break
		}
	}

	if supported == false {
		return fmt.Errorf("Architecture %s is not in the supported architecture list %v.", a, b.supported)
	}

	cmd := exec.Command("cowbuilder", "--create", "--buildplace", imagePath)
	cmd.Env = []string{fmt.Sprintf("HOME=%s", b.basepath)}
	cmd.Stdout = output
	cmd.Stderr = output
	cmd.Stdin = nil

	return cmd.Run()
}

func (b *Cowbuilder) RemoveDistribution(d deb.Distribution, a deb.Architecture) error {
	b.acquire()
	defer b.release()

	_, err := b.supportedDistributionPath(d, a)
	if err != nil {
		return fmt.Errorf("Distribution %s architecture %s is not supported", d, a)
	}

	return deb.NotYetImplemented()
}

func (b *Cowbuilder) UpdateDistribution(d deb.Distribution, a deb.Architecture) error {
	b.acquire()
	defer b.release()

	_, err := b.supportedDistributionPath(d, a)
	if err != nil {
		return fmt.Errorf("Distribution %s architecture %s is not supported", d, a)
	}

	return deb.NotYetImplemented()
}

func (b *Cowbuilder) AvailableDistributions() []deb.Distribution {
	b.acquire()
	defer b.release()

	// checks for
	res := []deb.Distribution{}
	for d, _ := range b.getAllImages() {
		res = append(res, d)
	}
	return res
}

func (b *Cowbuilder) AvailableArchitectures(d deb.Distribution) []deb.Architecture {
	b.acquire()
	defer b.release()

	dists := b.getAllImages()
	if dists == nil {
		return nil
	}

	return dists[d]
}

func (b *Cowbuilder) acquire() {
	_ = <-b.semaphore
}

func (b *Cowbuilder) release() {
	b.semaphore <- true
}

func (b *Cowbuilder) imagePath(d deb.Distribution, a deb.Architecture) string {
	return path.Join(b.imagepath, fmt.Sprintf("%s-%s", d, a))
}

func (b *Cowbuilder) getAllImages() map[deb.Distribution][]deb.Architecture {

	allFiles, err := ioutil.ReadDir(b.imagepath)
	if err != nil {
		return nil
	}

	res := map[deb.Distribution][]deb.Architecture{}
	rx := regexp.MustCompile(`([a-z]+)-([a-z0-9]+)`)
	for _, f := range allFiles {
		if f.IsDir() == false {
			continue
		}

		matches := rx.FindStringSubmatch(f.Name())

		if matches == nil {
			continue
		}

		baseCow, err := os.Stat(path.Join(b.imagepath, f.Name(), "base.cow"))
		if err != nil {
			continue
		}

		if baseCow.IsDir() == false {
			continue
		}

		dist := deb.Distribution(matches[1])
		arch := deb.Architecture(matches[2])
		res[dist] = append(res[dist], arch)

	}
	return res
}

func (b *Cowbuilder) supportedDistributionPath(d deb.Distribution, a deb.Architecture) (string, error) {
	res := b.imagePath(d, a)
	baseCowPath := path.Join(res, "base.cow")
	baseCow, err := os.Stat(baseCowPath)
	if err != nil {
		return "", err
	}

	if baseCow.IsDir() == false {
		return "", fmt.Errorf("%s is not a directory", baseCowPath)
	}
	return res, nil
}

func (b *Cowbuilder) getSupportedArchitectures() {
	switch runtime.GOARCH {
	case "amd64":
		b.supported = []deb.Architecture{deb.Amd64, deb.I386}
	case "386":
		b.supported = []deb.Architecture{deb.I386}
	case "arm":
		b.supported = []deb.Architecture{deb.Armel}
	}
}
