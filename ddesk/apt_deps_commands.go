package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	deb ".."
)

// AddAptDependencyCommand is a CLI command that add or modifies an
// apt repository depenendency
type AddAptDependencyCommand struct {
	Dists   []string `short:"D" long:"dist" description:"codename of distribution to add" required:"true"`
	Comps   []string `short:"C" long:"comp" description:"component of distribution to add"`
	KeyFile string   `short:"K" long:"key" description:"PGP Public key file for non PPA repository"`
}

// Execute implements the command.
func (x *AddAptDependencyCommand) Execute(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("Missing address for creating repository")
	}
	i, err := NewInteractor(options)
	if err != nil {
		return err
	}

	toAdd := make(map[deb.Codename][]deb.Component)
	for _, dd := range x.Dists {
		d := deb.Codename(dd)
		_, ok := deb.CodenameList[d]
		if ok == false {
			return fmt.Errorf("Unknown distribution %s", d)
		}
		toAdd[d] = make([]deb.Component, 0, len(x.Comps))
		for _, c := range x.Comps {
			toAdd[d] = append(toAdd[d], deb.Component(c))
		}
	}
	for _, addressOrID := range args {
		actual := i.ListDependencies()
		_, ok := actual[AptRepositoryID(addressOrID)]
		id := AptRepositoryID(addressOrID)
		if ok == false {
			if strings.HasPrefix(addressOrID, "ppa:") {
				id, err = i.CreatePPADependency(addressOrID)
				if err != nil {
					return err
				}
			} else {
				if len(x.KeyFile) == 0 {
					return fmt.Errorf("Missing PGP keyfile for creating dependency on %s", addressOrID)
				}
				f, err := os.Open(x.KeyFile)
				if err != nil {
					return err
				}
				id, err = i.CreateRemoteDependency(addressOrID, f)
				if err != nil {
					return err
				}
			}

		}

		err = i.EditRepository(id, toAdd, nil)
		if err != nil {
			return err
		}
		log.Printf("Added %s %s to %s", x.Dists, x.Comps, addressOrID)
	}
	return nil
}

// RemoveAptDependencyCommand is a CLI command that edit/removes an
// apt repository dependency.
type RemoveAptDependencyCommand struct {
	Dists []string `short:"D" long:"dist" description:"codename of distribution to remove"`
	Comps []string `short:"C" long:"comp" description:"component of distribution to remove"`
}

// Execute implements the command.
func (x *RemoveAptDependencyCommand) Execute(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("Missing repository identifier(s)")
	}

	i, err := NewInteractor(options)
	if err != nil {
		return err
	}

	for _, idStr := range args {
		id := AptRepositoryID(idStr)
		if len(x.Dists) == 0 && len(x.Comps) == 0 {
			err = i.RemoveDependency(id)
			if err != nil {
				return err
			}
			log.Printf("Removed dependency on %s", id)
			continue
		}

		data, ok := i.ListDependencies()[id]
		if ok == false {
			return fmt.Errorf("Unknown repository %s", id)
		}

		toRemove := make(map[deb.Codename][]deb.Component)
		var dists []deb.Codename
		if len(x.Dists) == 0 {
			for d := range data.Components {
				dists = append(dists, d)
			}
		} else {
			for _, dd := range x.Dists {
				d := deb.Codename(dd)
				if _, ok := deb.CodenameList[d]; ok == false {
					return fmt.Errorf("Unknown distribution %s", d)
				}
				dists = append(dists, d)
			}
		}

		for _, d := range dists {
			if len(x.Comps) == 0 {
				toRemove[d] = data.Components[d]
			} else {
				toRemove[d] = make([]deb.Component, 0, len(x.Comps))
				for _, c := range x.Comps {
					toRemove[d] = append(toRemove[d], deb.Component(c))
				}
			}
		}

		err = i.EditRepository(id, nil, toRemove)
		if err != nil {
			return err
		}
		log.Printf("Edited repository %s", id)
	}
	return nil
}

// ListAptDependencyCommand is CLI command that list the current apt
// repository dependencies.
//
// It will print all dependencies on os.Stdout.
type ListAptDependencyCommand struct {
}

// Execute implements the command.
func (x *ListAptDependencyCommand) Execute(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("list-dependencies does not take any arguments")
	}

	i, err := NewInteractor(options)
	if err != nil {
		return err
	}

	deps := i.ListDependencies()
	if len(deps) == 0 {
		fmt.Printf("There is no build dependency on apt repositories\n")
		return nil
	}
	fmtStr := "%30s | %10s | %10s | %30s\n"
	fmt.Printf(fmtStr, "Repository", "key", "codename", "components")
	fmt.Printf(fmtStr, "------------------------------",
		"----------",
		"----------",
		"------------------------------")
	for id, data := range deps {
		for dist, comps := range data.Components {
			toPrint := make([]string, 0, len(comps))
			for _, c := range comps {
				toPrint = append(toPrint, string(c))
			}

			fmt.Printf(fmtStr, id, data.KeyID, dist, strings.Join(toPrint, ","))

		}
	}

	return nil

}

func init() {
	parser.AddCommand("add-dependency",
		"add an apt repository for fetching build dependency",
		"add an apt repository for fetching build dependency",
		&AddAptDependencyCommand{})

	parser.AddCommand("remove-dependency",
		"remove an apt repository for fetching build dependency",
		"remove an apt repository dependency. If no -D or -C flags are passed, remove the dependency completely, otherwise limit it only to these components",
		&RemoveAptDependencyCommand{})

	parser.AddCommand("list-dependencies",
		"Lists apt repository dependencies",
		"Lists apt repository dependencies",
		&ListAptDependencyCommand{})
}
