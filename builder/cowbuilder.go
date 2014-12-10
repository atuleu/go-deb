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

	keepEnv     []string
	debianDists []string
	ubuntuDists []string
	semaphore   chan bool
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

	res.keepEnv = []string{"PATH"}

	res.ubuntuDists = []string{"lucid", "maverick", "natty", "oneiric", "precise",
		"quantal", "raring", "saucy", "trusty", "utopic", "vivid"}
	res.debianDists = []string{"sid", "squeeze", "wheezy", "jessie", "stretch",
		"buster", "unstable", "testing", "stable"}

	return res, nil
}

func (b *Cowbuilder) maskedEnviron() []string {
	var res []string = nil
	for _, key := range b.keepEnv {
		value := os.Getenv(key)
		if len(value) == 0 {
			continue
		}
		res = append(res, key+"="+value)
	}
	return res
}

func (b *Cowbuilder) BuildPackage(a BuildArguments, output io.Writer) (*BuildResult, error) {
	b.acquire()
	defer b.release()

	return nil, deb.NotYetImplemented()
}

// returns an equivalent of .pbuilderrc run
func (b *Cowbuilder) cowbuilderCommand(d deb.Distribution, a deb.Architecture, command string, args ...string) (*exec.Cmd, error) {

	isUbuntu, err := b.isSupportedUbuntu(d)
	if err != nil {
		return nil, err
	}

	imagePath := b.imagePath(d, a)
	baseCowPath := path.Join(imagePath, "base.cow")
	buildPath := path.Join(imagePath, "build")
	aptCache := path.Join(b.basepath, "images/aptcache")
	ccache := path.Join(b.basepath, "images/ccache")
	toCreate := []string{buildPath, aptCache, ccache}
	for _, d := range toCreate {
		err = os.MkdirAll(d, 0755)
		if err != nil {
			return nil, err
		}
	}
	preDebootstrapOpts := fmt.Sprintf("\"--arch\" \"%s\"", a)
	var mirror, components, mirrorsite, postDebootstrapOpts string
	if isUbuntu == true {
		mirror = "http://ftp.ubuntu.com/ubuntu"
		mirrorsite = "http://ftp.ubuntu.com/ubuntu"
		components = "main restricted universe multiverse"
		postDebootstrapOpts = "\"--keyring=/usr/share/keyrings/ubuntu-archive-keyring.gpg\""
	} else {
		mirror = "http://ftp.us.debian.org/debian"
		mirrorsite = "http://ftp.us.debian.org/debian"
		components = "main contrib non-free"
		postDebootstrapOpts = "\"--keyring=/usr/share/keyrings/debian-archive-keyring.gpg\""
	}

	cmd := exec.Command("cowbuilder", command)
	cmd.Args = append(cmd.Args, args...)

	cmd.Env = append(b.maskedEnviron(), fmt.Sprintf("HOME=%s", b.basepath))

	f, err := os.Create(path.Join(b.basepath, ".pbuilderrc"))
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(f, "%s=\"%s\"\n", "BASEPATH", baseCowPath)
	fmt.Fprintf(f, "%s=\"%s\"\n", "BUILDPLACE", buildPath)
	fmt.Fprintf(f, "%s=\"%s\"\n", "DISTRIBUTION", d)
	fmt.Fprintf(f, "%s=\"%s\"\n", "ARCHITECTURE", a)
	fmt.Fprintf(f, "%s=\"%s\"\n", "APTCACHE", aptCache)
	fmt.Fprintf(f, "%s=(%s \"${DEBOOTSTRAPOPTS[@]}\" %s)\n", "DEBOOTSTRAPOPTS", preDebootstrapOpts, postDebootstrapOpts)
	fmt.Fprintf(f, "%s=\"%s\"\n", "MIRROR", mirror)
	fmt.Fprintf(f, "%s=\"%s\"\n", "MIRRORSITE", mirrorsite)
	fmt.Fprintf(f, "%s=\"%s\"\n", "COMPONENTS", components)

	return cmd, nil
}

func (b *Cowbuilder) setBuildResult(cmd *exec.Cmd, path string) {
	cmd.Env = append(cmd.Env, fmt.Sprintf("BUILDRESULT=%s", path))
}

// Returns true if it is a supported ubuntu distribution, or false if
// it is a supported Debian one.
// if not supported returns an error
func (b *Cowbuilder) isSupportedUbuntu(d deb.Distribution) (bool, error) {
	for _, dd := range b.ubuntuDists {
		if d == deb.Distribution(dd) {
			return true, nil
		}
	}

	for _, dd := range b.debianDists {
		if d == deb.Distribution(dd) {
			return false, nil
		}
	}

	return false, fmt.Errorf("%s is not supported by this builder", d)
}

func (b *Cowbuilder) InitDistribution(d deb.Distribution, a deb.Architecture, output io.Writer) error {
	b.acquire()
	defer b.release()

	_, err := b.supportedDistributionPath(d, a)
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

	cmd, err := b.cowbuilderCommand(d, a, "--create")

	cmd.Stdout = output
	cmd.Stderr = output
	cmd.Stdin = nil
	if output != nil {
		fmt.Fprintf(output, "--- Executing: %v\n--- Env: %v\n", cmd.Args, cmd.Env)
	}
	return cmd.Run()
}

func (b *Cowbuilder) RemoveDistribution(d deb.Distribution, a deb.Architecture) error {
	b.acquire()
	defer b.release()

	imagePath, err := b.supportedDistributionPath(d, a)
	if err != nil {
		return fmt.Errorf("Distribution %s architecture %s is not supported", d, a)
	}

	return os.RemoveAll(imagePath)
}

func (b *Cowbuilder) UpdateDistribution(d deb.Distribution, a deb.Architecture, output io.Writer) error {
	b.acquire()
	defer b.release()

	_, err := b.supportedDistributionPath(d, a)
	if err != nil {
		return fmt.Errorf("Distribution %s architecture %s is not supported", d, a)
	}

	cmd, err := b.cowbuilderCommand(d, a, "--update")
	if err != nil {
		return err
	}
	cmd.Stdin = nil
	cmd.Stdout = output
	cmd.Stderr = output

	if output != nil {
		fmt.Fprintf(output, "--- Executing: %v\n--- Env: %v\n", cmd.Args, cmd.Env)
	}

	return cmd.Run()
}

func (b *Cowbuilder) AvailableDistributions() []deb.Distribution {
	b.acquire()
	defer b.release()

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
