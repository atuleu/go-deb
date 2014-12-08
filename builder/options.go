package main

import "github.com/jessevdk/go-flags"

// #include <sys/file.h>
import "C"

type Options struct {
	BuilderType   string `long:"type" short:"t" description:"Builder type for build operation, supported are client or cowbuilder" default:"client"`
	BuilderSocket string `long:"socket" short:"s" description:"For client builder, address of the rpc server" default:"/var/lib/go-deb.builder/builder.sock"`
}

var options = &Options{}

var parser = flags.NewParser(options, flags.Default)

func init() {
}
