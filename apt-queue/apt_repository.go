package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strings"

	deb ".."
	"github.com/nightlyone/lockfile"
)

type RepoDist struct {
	Components    []deb.Component
	Architectures []deb.Architecture
	Codename      deb.Codename
	Vendor        deb.Vendor
}

type AptRepo interface {
	Add(deb.Codename, []deb.Architecture, []deb.Component) error
	Remove(deb.Codename, []deb.Architecture, []deb.Component) error
	List() map[deb.Codename]RepoDist
	Include(string, deb.Codename, []deb.Component) error
}

type RepreproRepository struct {
	workingdir                           string
	confdir                              string
	distConfigPath                       string
	lock                                 lockfile.Lockfile
	dists                                map[deb.Codename]RepoDist
	Origin, Label, Description, SignWith string
	Components                           []deb.Component
}

func NewRepreproRepository(dir string, keyId string) (*RepreproRepository, error) {
	res := &RepreproRepository{
		workingdir: dir,
		dists:      make(map[deb.Codename]RepoDist),
	}

	res.confdir = path.Join(dir, "conf")
	res.distConfigPath = path.Join(res.workingdir, "conf/distributions")

	err := os.MkdirAll(res.confdir, 0755)
	if err != nil {
		return nil, err
	}

	conflockpath := path.Join(res.confdir, "conf.lock")

	res.lock, err = lockfile.New(conflockpath)
	if err != nil {
		return nil, err
	}

	err = res.load()
	if err != nil {
		return nil, fmt.Errorf("%s parse error: %s", res.distConfigPath, err)
	}

	return res, nil
}

func (r *RepreproRepository) tryLock() error {
	if err := r.lock.TryLock(); err != nil {
		return fmt.Errorf("Could not lock %s: %s", r.lock, err)
	}
	return nil
}

func (r *RepreproRepository) unlockOrPanic() {
	if err := r.lock.Unlock(); err != nil {
		panic(err)
	}
}

func (r *RepreproRepository) setField(name, value string) error {
	value = strings.TrimSpace(value)
	fieldValue := reflect.ValueOf(r).FieldByName(name)
	if len(fieldValue.String()) == 0 {
		fieldValue.SetString(value)
	}

	if fieldValue.String() != value {
		return fmt.Errorf("Could not set %s to %s, as it has value %s", name, value, fieldValue.String())
	}
	return nil
}

func (r *RepreproRepository) load() error {
	if err := r.tryLock(); err != nil {
		return err
	}
	defer r.unlockOrPanic()

	f, err := os.Open(r.distConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	l := deb.NewControlFileLexer(f)
	newDist := deb.Codename("")
	newArch := []deb.Architecture{}
	newComponents := []deb.Component{}
	newVendor := deb.Vendor("")
	stop := false
	for stop == false {
		field, err := l.Next()
		if err == io.EOF {
			stop = true
		} else if err != nil {
			return err
		}

		if deb.IsNewParagraph(field) {
			if len(newDist) == 0 {
				return fmt.Errorf("missing Codename:")
			}
			if len(newArch) == 0 {
				return fmt.Errorf("missing Architectures:")
			}
			if len(newComponents) == 0 {
				return fmt.Errorf("missing Components:")
			}

			r.dists[newDist] = RepoDist{
				Codename:      newDist,
				Components:    newComponents,
				Architectures: newArch,
				Vendor:        newVendor,
			}

			newDist = deb.Codename("")
			newComponents = nil
			newArch = nil
			newVendor = deb.Vendor("")
		}

		if len(field.Data) != 1 {
			return fmt.Errorf("invalide field %s, expected single line", field)
		}

		switch field.Name {
		case "Codename":
			cStr := strings.TrimSpace(field.Data[0])
			newDist = deb.Codename(cStr)
			var ok bool
			if newVendor, ok = deb.CodenameList[newDist]; ok == false {
				return fmt.Errorf("Unknow codename %s", cStr)
			}
		case "Architectures":
			for _, aStr := range strings.Split(field.Data[0], " ") {
				if len(aStr) == 0 {
					continue
				}
				arch := deb.Architecture(aStr)
				if _, ok := deb.ArchitectureList[arch]; ok == false {
					return fmt.Errorf("invalid architecture %s", arch)
				}
				newArch = append(newArch, arch)
			}
		case "Components":
			if len(newDist) == 0 {
				return fmt.Errorf("Missing previous Codename")
			}
			for _, cStr := range strings.Split(field.Data[0], " ") {
				if len(cStr) == 0 {
					continue
				}

				newComponents = append(newComponents, deb.Component(cStr))
			}
		case "Origin", "Label", "Description", "SignWith":
			if err := r.setField(field.Name, field.Data[0]); err != nil {
				return err
			}

		default:
			return fmt.Errorf("Unhandled field %s", field)
		}

	}

	return nil
}

func (r *RepreproRepository) save() error {
	if err := r.tryLock(); err != nil {
		return err
	}
	defer r.unlockOrPanic()

	f, err := os.Create(r.distConfigPath)
	if err != nil {
		return err
	}

	for _, d := range r.dists {
		fmt.Fprintf(f, "# %s/%s\n", d.Vendor, d.Codename)
		fmt.Fprintf(f, "Codename: %s\n", d.Codename)
		fmt.Fprintf(f, "Origin: %s\n", r.Origin)
		fmt.Fprintf(f, "Label: %s\n", r.Label)
		fmt.Fprintf(f, "Description: %s\n", r.Description)
		fmt.Fprintf(f, "SignWith: %s\n", r.SignWith)
		fmt.Fprintf(f, "Components:")
		for _, c := range d.Components {
			fmt.Fprintf(f, " %s", c)
		}
		fmt.Fprintf(f, "\nArchitectures:")
		for _, a := range d.Architectures {
			fmt.Fprintf(f, " %s", a)
		}
		fmt.Fprintf(f, "\n\n")
	}

	return nil
}

func (r *RepreproRepository) Add(d deb.Codename, archs []deb.Architecture, comps []deb.Component) error {
	toModify, ok := r.dists[d]
	if ok == true {
		existingArch := map[deb.Architecture]bool{}
		for _, a := range toModify.Architectures {
			existingArch[a] = true
		}
		for _, a := range archs {
			existingArch[a] = true
		}

		toModify.Architectures = make([]deb.Architecture, 0, len(existingArch))
		for a, _ := range existingArch {
			toModify.Architectures = append(toModify.Architectures, a)
		}

		existingComp := map[deb.Component]bool{}
		for _, c := range toModify.Components {
			existingComp[c] = true
		}

		for _, c := range comps {
			existingComp[c] = true
		}
		toModify.Components = make([]deb.Component, 0, len(existingComp))
		for c, _ := range existingComp {
			toModify.Components = append(toModify.Components, c)
		}
	} else {
		vendor, exists := deb.CodenameList[d]
		if exists == false {
			return fmt.Errorf("Unknow distribution codename %s", d)
		}
		r.dists[d] = RepoDist{
			Codename:      d,
			Vendor:        vendor,
			Components:    comps,
			Architectures: archs,
		}
	}

	if len(r.dists[d].Components) == 0 {
		return fmt.Errorf("Invalid %s definition: missing at least one component", d)
	}
	if len(r.dists[d].Architectures) == 0 {
		return fmt.Errorf("Invalid %s definition: missing at least one architecture", d)
	}

	saved := r.dists[d]

	if err := r.save(); err != nil {
		if ok == true {
			r.dists[d] = saved
		} else {
			delete(r.dists, d)
		}
		return err
	}

	return nil
}

func (r *RepreproRepository) Remove(d deb.Codename, archs []deb.Architecture, comps []deb.Component) error {
	toModify, ok := r.dists[d]
	if ok == false {
		return fmt.Errorf("Could not modify unlisted distribution %s", d)
	}

	curArchs := map[deb.Architecture]bool{}
	for _, a := range toModify.Architectures {
		curArchs[a] = true
	}
	for _, a := range archs {
		if _, ok := curArchs[a]; ok == false {
			return fmt.Errorf("%s does not list %s architecture", d, a)
		}
		delete(curArchs, a)
	}
	toModify.Architectures = make([]deb.Architecture, 0, len(curArchs))
	for a, _ := range curArchs {
		toModify.Architectures = append(toModify.Architectures, a)
	}

	curComps := map[deb.Component]bool{}
	for _, c := range toModify.Components {
		curComps[c] = true
	}
	for _, c := range comps {
		if _, ok := curComps[c]; ok == false {
			return fmt.Errorf("%s does not list %s component", d, c)
		}
		delete(curComps, c)
	}
	toModify.Components = make([]deb.Component, 0, len(curComps))
	for c, _ := range curComps {
		toModify.Components = append(toModify.Components, c)
	}

	saved := r.dists[d]
	if len(toModify.Architectures) == 0 || len(toModify.Components) == 0 {
		delete(r.dists, d)
	} else {
		r.dists[d] = toModify
	}

	if err := r.save(); err != nil {
		r.dists[d] = saved
		return err
	}

	return nil
}

func (r *RepreproRepository) List() map[deb.Codename]RepoDist {
	return r.dists
}

func (r *RepreproRepository) Include(changePath string, d deb.Codename, comps []deb.Component) error {
	if err := r.tryLock(); err != nil {
		return err
	}
	defer r.unlockOrPanic()

	dist, ok := r.dists[d]
	if ok == false {
		return fmt.Errorf("Distribution %s is not supported", d)
	}

	if len(comps) == 0 {
		comps = []deb.Component{"all"}
	}

	for _, c := range comps {
		var cmd *exec.Cmd = nil
		if c == "all" {
			cmd = exec.Command("reprepro", "include", string(dist.Codename), changePath)
		} else {
			ok = false
			for _, supportedC := range dist.Components {
				if c == supportedC {
					ok = true
					break
				}
			}
			if ok == false {
				return fmt.Errorf("Distribution %s does not list component %s", d, c)
			}

			cmd = exec.Command("reprepro", "-C", string(c), "include", string(dist.Codename), changePath)
		}

		cmd.Dir = r.workingdir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Could not include %s: %s\n%s", changePath, err, out)
		}
	}

	return nil
}
