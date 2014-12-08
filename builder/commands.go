package main

import (
	"fmt"
	"os"
	"path"

	deb ".."
)

type ServeBuilderCommand struct {
	BasePath string `long:"basepath" short:"b" description:"basepath for the builder to run" default:"/var/lib/go-deb.builder"`
	Socket   string `long:"socket" short:"s" description:"socket relative to basepath" default:"builder.sock"`
	Type     string `long:"type" short:"t" description:"type of the builder" default:"cowbuilder"`
}

func (x *ServeBuilderCommand) Execute(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("take no arguments")
	}

	if x.Type != "cowbuilder" {
		return fmt.Errorf("Only --type=cowbuilder is supported")
	}

	b, err := NewCowbuilder(x.BasePath)
	if err != nil {
		return fmt.Errorf("Cowbuilder initialization error: %s", err)
	}

	socketPath := path.Join(x.BasePath, x.Socket)
	s := NewRpcBuilderServer(b, socketPath)
	// in any case we will remove the path

	go func() {
		err = s.WaitEstablished()
		if err != nil {
			panic(err)
		}
	}()
	s.Serve()
	return nil
}

type InitDistributionCommand struct {
	Dist string `long:"dist" short:"D" description:"Distribution to initilaize" required:"true"`
	Arch string `long:"arch" short:"A" description:"Architecture to initialize" required:"true"`
}

func (x *InitDistributionCommand) Execute(args []string) error {
	i, err := NewInteractor(options)
	if err != nil {
		return err
	}

	res, err := i.AddDistributionSupport(deb.Distribution(x.Dist),
		deb.Architecture(x.Arch),
		os.Stdout)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", res.Message)
	return nil

}

type RemoveDistributionCommand struct {
	Dist        string `long:"dist" short:"D" description:"Distribution to initilaize" required:"true"`
	Arch        string `long:"arch" short:"A" description:"Architecture to initialize" required:"true"`
	RemoveCache bool   `long:"remove-cache" description:"Remove cached data by the builder"`
}

func (x *RemoveDistributionCommand) Execute(args []string) error {
	i, err := NewInteractor(options)
	if err != nil {
		return err
	}

	err = i.RemoveDistributionSupport(deb.Distribution(x.Dist),
		deb.Architecture(x.Arch),
		x.RemoveCache)
	if err != nil {
		return err
	}

	fmt.Printf("Removed user distribution support for %s-%s\n", x.Dist, x.Arch)
	return nil

}

func init() {
	parser.AddCommand("serve-builder",
		"Starts a package builder as a RPC service.",
		"Starts a builder that can build debian packages via a RPC service. Please note that this service would only be available locally.",
		&ServeBuilderCommand{})

	parser.AddCommand("init-dist",
		"Init a new distribution / architecture",
		"Inits a new distribution / architecture",
		&InitDistributionCommand{})

	parser.AddCommand("remove-dist",
		"Remove support for a  distribution / architecture couple",
		"Remove support for a  distribution / architecture couple. Please note that without the --remove-cache, it will not remove any cached data by the actual builder, but just edit the user settings.",
		&InitDistributionCommand{})

}
