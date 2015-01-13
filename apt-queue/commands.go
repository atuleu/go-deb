package main

import (
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

type AddDistributionCommand struct {
	Dist string `long:"dist" short:"D" description:"Distribution to add" required:"true"`
	Arch string `long:"arch" short:"A" description:"Architecture to add" required:"true"`
}

func (x *AddDistributionCommand) Execute(args []string) error {
	return deb.NotYetImplemented()
}

type RemoveDistributionCommand struct {
	Dist string `long:"dist" short:"D" description:"Distribution to add" required:"true"`
	Arch string `long:"arch" short:"A" description:"Architecture to add" required:"false"`
}

func (x *RemoveDistributionCommand) Execute(args []string) error {
	return deb.NotYetImplemented()
}

type ListDistributionCommand struct {
}

func (x *ListDistributionCommand) Execute(args []string) error {
	return deb.NotYetImplemented()
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
		&AddDistributionCommand{})

	parser.AddCommand("remove",
		"Removes distribution /  architecture",
		"Removes a couple distribution / architecture from the archive",
		&RemoveDistributionCommand{})

	parser.AddCommand("list",
		"Lists archived distributions and architectures",
		"Lists archived distributions and architectures",
		&ListDistributionCommand{})
}
