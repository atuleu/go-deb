package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	deb ".."
	"github.com/nightlyone/lockfile"
	"launchpad.net/go-xdg"
)

type LocalReprepro struct {
	dists map[deb.Distribution][]deb.Architecture
	lock  lockfile.Lockfile

	basepath    string
	distribpath string
}

func (r *LocalReprepro) tryLock() error {
	if err := r.lock.TryLock(); err != nil {
		return fmt.Errorf("Could not lock repository %s: %s", r.basepath, err)
	}
	return nil
}

func (r *LocalReprepro) unlockOrPanic() {
	if err := r.lock.Unlock(); err != nil {
		panic(fmt.Sprintf("Could not unlock %s: %s", r.basepath, err))
	}
}

func (r *LocalReprepro) loadDistributions() error {
	if err := r.tryLock(); err != nil {
		return err
	}
	defer r.unlockOrPanic()

	f, err := os.Open(r.distribpath)
	if err != nil {
		return fmt.Errorf("Could not open distribution configuration")
	}

	r.dists = make(map[deb.Distribution][]deb.Architecture)

	reader := bufio.NewReader(f)

	curDist := deb.Distribution("")
	reachedEof := false
	for reachedEof == false {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				reachedEof = true
			} else {
				return fmt.Errorf("Could not read conf/distributions file: %s", err)
			}
		}

		line = strings.TrimSpace(line)
		if len(line) == 0 {
			curDist = deb.Distribution("")
			continue
		}

		if strings.HasPrefix(line, "Codename:") {
			curDist = deb.Distribution(strings.TrimSpace(strings.TrimPrefix(line, "Codename:")))
			continue
		}

		if strings.HasPrefix(line, "Architectures:") {
			if len(curDist) == 0 {
				return fmt.Errorf("conf/distributions parse error, found 'Architectures:' without prior 'Codename:'")
			}

			archs := strings.Split(strings.TrimPrefix(line, "Architectures:"), " ")
			for _, confArch := range archs {
				a := deb.Architecture(confArch)
				switch a {
				case deb.Architecture("source"):
					continue
				case deb.Amd64, deb.I386, deb.Armel:
					r.dists[curDist] = append(r.dists[curDist], a)
				default:
					return fmt.Errorf("conf/distribution parse error, found invalid architecture `%s'", confArch)
				}
			}
		}
	}

	return nil
}

func (r *LocalReprepro) writeDistributions() error {
	if err := r.tryLock(); err != nil {
		return err
	}
	defer r.unlockOrPanic()

	f, err := os.Create(r.distribpath)
	if err != nil {
		return fmt.Errorf("Could not open conf/distributions: %s", err)
	}
	defer f.Close()

	for d, archs := range r.dists {
		fmt.Fprintf(f, "Codename: %s\n", d)
		fmt.Fprintf(f, "Origin: Local builder rpository\n")
		fmt.Fprintf(f, "Label: Local builder rpository\n")
		fmt.Fprintf(f, "Label: Description\n")
		fmt.Fprintf(f, "Components: main\n")
		fmt.Fprintf(f, "SignWith: no\n")
		fmt.Fprintf(f, "Architectures:")
		for _, a := range archs {
			fmt.Fprintf(f, " %s", a)
		}
		fmt.Fprintf(f, "\n\n")
	}
	return nil
}

var lrBasepath = "go-deb.builder/local_apt"
var lrConfpath = path.Join(lrBasepath, "conf")
var lrDistributionConfig = path.Join(lrConfpath, "distributions")

func NewLocalReprepro() (*LocalReprepro, error) {
	distConfig, err := xdg.Data.Ensure(lrDistributionConfig)
	if err != nil {
		return nil, err
	}

	res := &LocalReprepro{
		distribpath: distConfig,
		basepath:    path.Dir(path.Dir(distConfig)),
	}

	res.lock, err = lockfile.New(path.Join(res.basepath, "reprepro.lock"))
	if err != nil {
		return nil, err
	}

	err = res.loadDistributions()
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (r *LocalReprepro) ArchiveBuildResult(b *BuildResult) error {
	targetDist := b.Changes.Dist
	if _, ok := r.dists[targetDist]; ok == false {
		return fmt.Errorf("Distribution %s is not supported", targetDist)
	}
	if err := r.tryLock(); err != nil {
		return err
	}
	defer r.unlockOrPanic()

	buildPackages, err := b.Changes.BinaryPackages()
	if err != nil {
		return fmt.Errorf("Could not get build packages list for %s: %s", b.Changes.Ref, err)
	}

	allPackages, err := r.unsafeListPackages(targetDist)
	if err != nil {
		return err
	}

	for _, b := range buildPackages {
		if _, ok := allPackages[b]; ok == false {
			continue
		}
		if err = r.unsafeRemovePackage(string(targetDist), b.Name); err != nil {
			return err
		}
	}

	cmd := exec.Command("reprepro",
		"include",
		string(targetDist),
		path.Join(b.BasePath, b.Changes.Ref.Filename()))

	cmd.Dir = r.basepath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Could not archive result of %s build:\n %s", b.Changes.Ref.Filename(), output)
	}

	return nil
}

func (r *LocalReprepro) AddDistribution(d deb.Distribution, a deb.Architecture) error {
	r.dists[d] = append(r.dists[d], a)
	return r.writeDistributions()
}

func (r *LocalReprepro) RemoveDistribution(d deb.Distribution, a deb.Architecture) error {
	archs, ok := r.dists[d]
	if ok == false {
		return fmt.Errorf("Distribution %s is not supported", d)
	}
	newArchs := []deb.Architecture{}
	found := false
	for _, aa := range archs {
		if aa == a {
			found = true
			continue
		}
		newArchs = append(newArchs, aa)
	}

	if found == false {
		return fmt.Errorf("Distribution %s does not support architecture %s", d, a)
	}

	if len(newArchs) != 0 {
		r.dists[d] = newArchs
	} else {
		delete(r.dists, d)
	}

	return r.writeDistributions()
}

func (r *LocalReprepro) unsafeListPackages(d deb.Distribution) (map[deb.BinaryPackageRef]bool, error) {
	if _, ok := r.dists[d]; ok == false {
		return nil, fmt.Errorf("Distribution %s is not supported", d)
	}

	var output bytes.Buffer
	cmd := exec.Command("reprepro", "list", string(d))
	cmd.Dir = r.basepath
	cmd.Stderr = &output
	cmd.Stdout = &output
	cmd.Stdin = nil

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("reprepro could not list %s:\n%s", d, output.String())
	}

	res := make(map[deb.BinaryPackageRef]bool)
	eofReached := false
	packRx := regexp.MustCompile(`^.*|.*|(.*): (.*) (.*)$`)
	for eofReached == false {
		l, err := output.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				eofReached = true
			} else {
				return nil, fmt.Errorf("Could not parse reprepro list output: %s", err)
			}
		}

		matches := packRx.FindStringSubmatch(l)
		if matches == nil {
			return nil, fmt.Errorf("Could not parse reprepro list output line `%s', it does not match %s",
				strings.TrimSpace(l),
				packRx)
		}

		ver, err := deb.ParseVersion(matches[3])
		if err != nil {
			return nil, err
		}

		if matches[1] == "source" {
			continue
		}

		p := deb.BinaryPackageRef{
			Name: matches[2],
			Ver:  *ver,
			Arch: deb.Architecture(matches[1]),
		}
		res[p] = true
	}
	return res, nil
}

func (r *LocalReprepro) ListPackage(d deb.Distribution, rx *regexp.Regexp) []deb.BinaryPackageRef {
	if err := r.tryLock(); err != nil {
		return nil
	}
	defer r.unlockOrPanic()

	allPackages, _ := r.unsafeListPackages(d)
	res := make([]deb.BinaryPackageRef, 0, len(allPackages))
	for p, _ := range allPackages {
		if rx.MatchString(p.Name) {
			res = append(res, p)
		}
	}
	return res
}

func (r *LocalReprepro) unsafeRemovePackage(dist, pack string) error {
	cmd := exec.Command("reprepro", "remove", dist, pack)
	cmd.Dir = r.basepath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Could not remove package %s of distribution %s, reprepro returned:\n%s", pack, dist, output)
	}
	return nil
}

func (r *LocalReprepro) RemovePackage(d deb.Distribution, p deb.BinaryPackageRef) error {
	if _, ok := r.dists[d]; ok == false {
		return fmt.Errorf("Distributions %s is not supported", d)
	}
	if err := r.tryLock(); err != nil {
		return err
	}
	defer r.unlockOrPanic()

	return r.unsafeRemovePackage(string(d), p.Name)
}

func init() {
	aptDepTracker.Add("reprepro")
}
