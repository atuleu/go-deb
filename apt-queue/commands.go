package main

import (
	"fmt"

	deb ".."
)

type AddKeyCommand struct{}

func (x *AddKeyCommand) Execute(args []string) error {
	return deb.NotYetImplemented()
}

type RemoveKeyCommand struct{}

func (x *RemoveKeyCommand) Execute(args []string) error {
	return deb.NotYetImplemented()
}

type ListKeyCommand struct{}

func (x *ListKeyCommand) Execute(args []string) error {
	return deb.NotYetImplemented()
}

type DistributionManipCommand struct {
	Dists []string `long:"dist" short:"D" description:"Distribution(s) to add" required:"true"`
	Archs []string `long:"arch" short:"A" description:"Architecture(s) to add"`
	Comps []string `long:"component" short:"C" description:"Components(s) to add"`
	name  string
	op    func(*Interactor, deb.Codename, []deb.Architecture, []deb.Component) error
}

func (x *DistributionManipCommand) Execute(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("%s command takes no arguments", x.name)
	}

	i, err := NewInteractor(options)
	if err != nil {
		return err
	}
	archs := make([]deb.Architecture, 0, len(x.Archs))
	comps := make([]deb.Component, 0, len(x.Comps))
	for _, a := range x.Archs {
		arch, err := deb.ParseArchitecture(a)
		if err != nil {
			return err
		}
		archs = append(archs, arch)
	}
	for _, c := range x.Comps {
		comps = append(comps, deb.Component(c))
	}

	for _, d := range x.Dists {
		if err = x.op(i, deb.Codename(d), archs, comps); err != nil {
			return err
		}
	}
	return nil
}

type ListDistributionCommand struct {
}

func (x *ListDistributionCommand) Execute(args []string) error {
	i, err := NewInteractor(options)
	if err != nil {
		return err
	}
	dists := i.ListDistributions()
	if len(dists) == 0 {
		fmt.Printf("There is no distribution supported\n")
		return nil
	}

	fmtStr := "%20s | %10s | %20s\n"
	fmt.Printf(fmtStr, "Codename", "Architecture", "Component")
	fmt.Printf(fmtStr, "--------------------", "----------", "--------------------")
	for _, d := range dists {
		for _, a := range d.Architectures {
			for _, c := range d.Components {
				fmt.Printf(fmtStr, d.Codename, a, c)
			}
		}
	}

	return nil
}

func init() {
	parser.AddCommand("add-key",
		"Authorize the given key to upload to this repositories",
		"Takes exactly one argument, the path to the key to add.",
		&AddKeyCommand{})

	parser.AddCommand("remove-key",
		"Remove a key from the list of authorized keys",
		"After this step, uploaded packet signed with this key will not be authorized to be archived",
		&RemoveKeyCommand{})

	parser.AddCommand("list-keys",
		"Lists all authorized keys",
		"Lists all authorized keys",
		&ListKeyCommand{})

	parser.AddCommand("add",
		"Add distribution /  architecture",
		"Adds a couple distribution / architecture from the archive",
		&DistributionManipCommand{
			op: func(i *Interactor, d deb.Codename, a []deb.Architecture, c []deb.Component) error {
				return i.AddDistribution(d, a, c)
			},
			name: "add",
		})

	parser.AddCommand("remove",
		"Removes distribution /  architecture",
		"Removes a couple distribution / architecture from the archive",
		&DistributionManipCommand{
			op: func(i *Interactor, d deb.Codename, a []deb.Architecture, c []deb.Component) error {
				return i.RemoveDistribution(d, a, c)
			},
			name: "remove",
		})

	parser.AddCommand("list",
		"Lists archived distributions and architectures",
		"Lists archived distributions and architectures",
		&ListDistributionCommand{})
}
