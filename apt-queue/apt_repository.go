package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"strings"

	deb ".."
	"github.com/nightlyone/lockfile"
)

type RepoDist struct {
	Components    []deb.Component
	Architectures []deb.Architecture
	Codename      string
}

type AptRepo interface {
	Add(deb.Distribution, []deb.Architecture, []deb.Component) error
	Remove(deb.Distribution, []deb.Architecture, []deb.Component) error
	List() map[string]RepoDist
	Include(*deb.ChangesFile, []deb.Component) error
}

type RepreproRepository struct {
	workingdir                           string
	confdir                              string
	distConfigPath                       string
	lock                                 lockfile.Lockfile
	dists                                map[string]RepoDist
	Origin, Label, Description, SignWith string
	Components                           []deb.Component
}

func NewRepreproRepository(dir string, keyId string) (*RepreproRepository, error) {
	res := &RepreproRepository{
		workingdir: dir,
		dists:      make(map[string]RepoDist),
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
	newDist := ""
	newArch := []deb.Architecture{}
	newComponents := []deb.Component{}

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
			}

			newDist = ""
			newComponents = nil
			newArch = nil
		}

		if len(field.Data) != 1 {
			return fmt.Errorf("invalide field %s, expected single line", field)
		}

		switch field.Name {
		case "Codename":
			newDist = field.Data[0]
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
		fmt.Fprintf(f, "# %s/%s\n", "foo", d.Codename)
		fmt.Fprintf(f, "Origin: %s\n", r.Origin)
		fmt.Fprintf(f, "Label: %s\n", r.Label)
		fmt.Fprintf(f, "Description: %s\n", r.Description)
		fmt.Fprintf(f, "SignWith: %s\n", r.SignWith)
		fmt.Fprintf(f, "Codename: %s\n", d.Codename)
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

func (r *RepreproRepository) Add(deb.Distribution, []deb.Architecture, []deb.Component) error {
	return deb.NotYetImplemented()
}
func (r *RepreproRepository) Remove(deb.Distribution, []deb.Architecture, []deb.Component) error {
	return deb.NotYetImplemented()
}
func (r *RepreproRepository) List() map[string]RepoDist {
	return nil
}
func (r *RepreproRepository) Include(*deb.ChangesFile, []deb.Component) error {
	return deb.NotYetImplemented()
}
