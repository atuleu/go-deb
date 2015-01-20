package main

import "github.com/jessevdk/go-flags"

type Options struct {
	Base string `short:"b" long:"basepath" description:"Basepath on the system" default:"/var/lib/go-deb.apt-repo-queue"`
}

var options = &Options{}

var parser = flags.NewParser(options, flags.Default)

func init() {
}
