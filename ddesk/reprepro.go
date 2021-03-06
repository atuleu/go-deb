package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	deb ".."
	"github.com/nightlyone/lockfile"
)

type Reprepro struct {
	dists map[deb.Codename]map[deb.Architecture]bool
	lock  lockfile.Lockfile

	basepath string
}

func (r *Reprepro) tryLock() error {
	if err := r.lock.TryLock(); err != nil {
		return fmt.Errorf("Could not lock repository %s: %s", r.basepath, err)
	}
	return nil
}

func (r *Reprepro) unlockOrPanic() {
	if err := r.lock.Unlock(); err != nil {
		panic(fmt.Sprintf("Could not unlock %s: %s", r.basepath, err))
	}
}

func (r *Reprepro) loadDistributions() error {
	if err := r.tryLock(); err != nil {
		return err
	}
	defer r.unlockOrPanic()

	f, err := os.Open(r.distributionsConfig())
	if err != nil {
		return err
	}

	r.dists = make(map[deb.Codename]map[deb.Architecture]bool)

	l := deb.NewControlFileLexer(f)
	newDist := deb.Codename("")
	newArch := []deb.Architecture{}

	for {
		field, err := l.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if deb.IsNewParagraph(field) {
			if len(newDist) != 0 && len(newArch) == 0 {
				return fmt.Errorf("paragraph with Codename: but no Architectures: field")
			}
			if len(newArch) != 0 && len(newDist) == 0 {
				return fmt.Errorf("paragraph with Architectures: but no Codename: field")
			}
			continue
		}

		if len(field.Data) != 1 {
			return fmt.Errorf("expect single line field only")
		}

		if field.Name == "Codename" {
			if strings.Contains(field.Data[0], " ") == true {
				return fmt.Errorf("Invalid Codename: field %s", field.Data[0])
			}
			newDist = deb.Codename(field.Data[0])
		}

		if field.Name == "Architectures" {
			for _, aStr := range strings.Split(field.Data[0], " ") {
				newArch = append(newArch, deb.Architecture(aStr))
			}
		}

		if len(newDist) != 0 && len(newArch) != 0 {
			r.dists[newDist] = make(map[deb.Architecture]bool)
			for _, a := range newArch {
				r.dists[newDist][a] = true
			}
			newDist = ""
			newArch = nil
		}
	}

	return nil
}

func (r *Reprepro) writeDistributions() error {
	if err := r.tryLock(); err != nil {
		return err
	}
	defer r.unlockOrPanic()

	f, err := os.Create(r.distributionsConfig())
	if err != nil {
		return err
	}
	defer f.Close()

	for d, archs := range r.dists {
		fmt.Fprintf(f, "Codename: %s\n", d)
		fmt.Fprintf(f, "Origin: Local ddesk repository\n")
		fmt.Fprintf(f, "Label: Local ddesk repository\n")
		fmt.Fprintf(f, "Description: Local ddesk repository\n")
		fmt.Fprintf(f, "Components: main\n")
		fmt.Fprintf(f, "Architectures:")
		for a, _ := range archs {
			fmt.Fprintf(f, " %s", a)
		}
		fmt.Fprintf(f, "\n\n")
	}
	return nil
}

func (r *Reprepro) confPath() string {
	return path.Join(r.basepath, "conf")
}

func (r *Reprepro) distributionsConfig() string {
	return path.Join(r.confPath(), "distributions")
}

func NewReprepro(basepath string) (*Reprepro, error) {
	res := &Reprepro{
		basepath: basepath,
	}

	if _, err := os.Stat(res.distributionsConfig()); err != nil {
		if os.IsNotExist(err) == false {
			return nil, fmt.Errorf("Could not check existence of %s: %s",
				res.distributionsConfig(), err)
		}
		err = os.MkdirAll(path.Dir(res.distributionsConfig()), 0755)
		if err != nil {
			return nil, err
		}
		f, err := os.Create(res.distributionsConfig())
		if err != nil {
			return nil, err
		}
		f.Close()
	}
	var err error

	res.lock, err = lockfile.New(path.Join(res.basepath, "reprepro.lock"))
	if err != nil {
		return nil, err
	}

	err = res.loadDistributions()
	if err != nil {
		return nil, fmt.Errorf("conf/distributions parse error: %s", err)
	}

	return res, nil
}

func (r *Reprepro) ArchiveChanges(c *deb.ChangesFile, dir string) error {
	targetDist := c.Dist
	if _, ok := r.dists[targetDist]; ok == false {
		return fmt.Errorf("Distribution %s is not supported", targetDist)
	}
	if err := r.tryLock(); err != nil {
		return err
	}
	defer r.unlockOrPanic()

	buildPackages, err := c.BinaryPackages()
	if err != nil {
		return fmt.Errorf("Could not get build packages list for %s: %s", c.Ref, err)
	}

	allPackages, err := r.unsafeListPackages(targetDist)
	if err != nil {
		return fmt.Errorf("Could not list %s packages: %s", targetDist, err)
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
		path.Join(dir, c.Ref.Filename()))

	cmd.Dir = r.basepath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Could not archive result of %s build:\n %s", c.Ref.Filename(), output)
	}

	return nil
}

func (r *Reprepro) AddDistribution(d deb.Codename, a deb.Architecture) error {
	saved, ok := r.dists[d]
	if ok == false {
		r.dists[d] = make(map[deb.Architecture]bool)
	}
	r.dists[d][a] = true
	if err := r.writeDistributions(); err != nil {
		if ok == false {
			delete(r.dists, d)
		} else {
			r.dists[d] = saved
		}
		return err
	}

	cmd := exec.Command("reprepro", "export", string(d))
	cmd.Dir = r.basepath
	cmdOut, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Could not bootstrap distribution %s: %s\n%s", d, err, cmdOut)
	}
	return nil
}

func (r *Reprepro) RemoveDistribution(d deb.Codename, a deb.Architecture) error {
	_, ok := r.dists[d]
	if ok == false {
		return fmt.Errorf("Distribution %s is not supported", d)
	}

	_, found := r.dists[d][a]

	if found == false {
		return fmt.Errorf("Distribution %s does not support architecture %s", d, a)
	}
	delete(r.dists[d], a)

	err := r.writeDistributions()
	if err != nil {
		r.dists[d][a] = true
		return err
	}
	return nil
}

func (r *Reprepro) unsafeListPackages(d deb.Codename) (map[deb.BinaryPackageRef]bool, error) {
	if _, ok := r.dists[d]; ok == false {
		return nil, fmt.Errorf("Distribution %s is not supported", d)
	}

	var output bytes.Buffer
	cmd := exec.Command("reprepro", "--list-format", "${package} ${version} ${architecture}\n", "list", string(d))
	cmd.Dir = r.basepath
	cmd.Stderr = &output
	cmd.Stdout = &output
	cmd.Stdin = nil

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("reprepro could not list %s:\n%s", d, output.String())
	}

	res := make(map[deb.BinaryPackageRef]bool)
	eofReached := false
	packRx := regexp.MustCompile(`^([a-z0-9][a-z0-9\+\-\.]+) ([^ ]+) ([^ ]+)\n$`)
	for eofReached == false {
		line, err := output.ReadString('\n')

		if err != nil {
			if err == io.EOF {
				eofReached = true
			} else {
				return nil, fmt.Errorf("Could not parse reprepro list output: %s", err)
			}
		}

		matches := packRx.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		ver, err := deb.ParseVersion(matches[2])
		if err != nil {
			return nil, err
		}

		p := deb.BinaryPackageRef{
			Name: matches[1],
			Ver:  *ver,
			Arch: deb.Architecture(matches[3]),
		}
		res[p] = true
	}
	return res, nil
}

func (r *Reprepro) ListPackage(d deb.Codename, rx *regexp.Regexp) []deb.BinaryPackageRef {
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

func (r *Reprepro) unsafeRemovePackage(dist, pack string) error {
	cmd := exec.Command("reprepro", "remove", dist, pack)
	cmd.Dir = r.basepath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Could not remove package %s of distribution %s, reprepro returned:\n%s", pack, dist, output)
	}
	return nil
}

func (r *Reprepro) RemovePackage(d deb.Codename, p deb.BinaryPackageRef) error {
	if _, ok := r.dists[d]; ok == false {
		return fmt.Errorf("Distributions %s is not supported", d)
	}
	if err := r.tryLock(); err != nil {
		return err
	}
	defer r.unlockOrPanic()

	return r.unsafeRemovePackage(string(d), p.Name)
}

func (r *Reprepro) Access() *AptRepositoryAccess {
	dists := make(map[deb.Codename][]deb.Component)
	for d, _ := range r.dists {
		dists[d] = []deb.Component{"main"}
	}
	absPath, _ := filepath.Abs(r.basepath)
	return &AptRepositoryAccess{
		ID:               AptRepositoryID(fmt.Sprintf("local:%s", absPath)),
		Components:       dists,
		Address:          fmt.Sprintf("file:%s", absPath),
		ArmoredPublicKey: nil,
	}
}

func init() {
	aptDepTracker.Add("reprepro")
}
