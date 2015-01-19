package main

import (
	"fmt"
	"os"
	"path"

	deb ".."
)

type ServeBuilderCommand struct {
	BasePath string `long:"basepath" short:"b" description:"basepath for the builder to run" default:"/var/lib/go-deb.ddesk"`
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

	// WRANING : do not create an intercator here. We could mess up
	// with user settings. we should just wrap a builder with a RpcBuilder.

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
	if len(args) > 0 {
		return fmt.Errorf("take no arguments")
	}

	i, err := NewInteractor(options)
	if err != nil {
		return err
	}

	res, err := i.AddDistributionSupport(deb.Codename(x.Dist),
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
	if len(args) > 0 {
		return fmt.Errorf("take no arguments")
	}

	i, err := NewInteractor(options)
	if err != nil {
		return err
	}

	err = i.RemoveDistributionSupport(deb.Codename(x.Dist),
		deb.Architecture(x.Arch),
		x.RemoveCache)
	if err != nil {
		return err
	}

	fmt.Printf("Removed user distribution support for %s-%s\n", x.Dist, x.Arch)
	return nil

}

type ListDistributionCommand struct{}

func (x *ListDistributionCommand) Execute(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("take no arguments")
	}

	i, err := NewInteractor(options)
	if err != nil {
		return err
	}

	distSupports, err := i.GetSupportedDistribution()
	if err != nil {
		return err
	}
	lineFormat := "%20s | %10s | %16s\n"
	fmt.Printf(lineFormat, "codename", "arch", "user supported")
	fmt.Printf("----------------------------------------------------\n")
	for d, archs := range distSupports {
		for a, userSupported := range archs {
			var toPrint string
			if userSupported == true {
				toPrint = "builder,user"
			} else {
				toPrint = "builder"
			}

			fmt.Printf(lineFormat, d, a, toPrint)
		}
	}
	return nil

}

type UpdateDistCommand struct {
	Dist string `long:"dist" short:"D" description:"Distribution to initilaize" required:"true"`
	Arch string `long:"arch" short:"A" description:"Architecture to initialize" required:"true"`
}

func (x *UpdateDistCommand) Execute(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("update-dist takes no arguments")
	}
	i, err := NewInteractor(options)
	if err != nil {
		return err
	}

	return i.builder.UpdateDistribution(deb.Codename(x.Dist),
		deb.Architecture(x.Arch),
		os.Stdout)
}

type BuildCommand struct {
}

func (x *BuildCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("build takes exactly one argument, the .dsc to build")
	}

	if err := deb.IsDscFileName(args[0]); err != nil {
		return fmt.Errorf("Invalid argument %s: %s", args[0], err)
	}
	f, err := os.Open(args[0])
	if err != nil {
		return err
	}

	i, err := NewInteractor(options)
	if err != nil {
		return err
	}
	// we use a stub auth to remove the signature
	cleared, err := i.auth.CheckAnyClearsigned(f)
	if err != nil {
		return err
	}

	dsc, err := deb.ParseDsc(cleared)
	if err != nil {
		return err
	}

	dsc.BasePath = path.Dir(args[0])

	res, err := i.BuildPackage(*dsc, os.Stdout)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully build %s, final changes is: %s\n", res.Changes.Ref.Identifier, path.Join(res.BasePath, res.ChangesPath))

	return nil
}

type InitInstallCommand struct{}

func (x *InitInstallCommand) Execute(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("install takes only one argument")
	}

	return aptDepTracker.SatisfyAll()
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
		&RemoveDistributionCommand{})

	parser.AddCommand("list-dist",
		"List current user and builder support for distribution/ architecture",
		"List current user and builder support for distribution/ architecture",
		&ListDistributionCommand{})

	parser.AddCommand("update-dist",
		"Update a supported distrbution",
		"Updates the chroot of any distribution of the builder.",
		&UpdateDistCommand{})

	parser.AddCommand("build",
		"Builds a .dsc file",
		"build will start a build for all architecture the user supports given a .dsc file. Distribution will be infered from the debian/changelog",
		&BuildCommand{})

	parser.AddCommand("install",
		"Install necessary files to the system",
		"Installs all necessary files to the system, likes package dependency, groups, and services",
		&InitInstallCommand{})
}
