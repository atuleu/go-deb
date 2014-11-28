package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"syscall"

	"github.com/jessevdk/go-flags"
)

// #include <sys/file.h>
import "C"

type Options struct {
	BaseDir  string                 `json:"-"`
	BaseFlag func(arg string) error `json:"-" short:"b" long:"basedir" description:"Base autobuild directory" default:"/var/lib/autobuild"`

	MaBite string `json:"ma-bite" short:"m" long:"ma-bite" description:"Ou elle est ma bite"`
}

func (o *Options) confPath() string {
	return path.Join(o.BaseDir, "etc/config.json")
}

func (o *Options) Load() error {
	f, err := os.Open(o.confPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("Could not open config file `%s' : %s", o.confPath(), err)
	}
	defer f.Close()

	err = syscall.Flock(int(f.Fd()), C.LOCK_EX)
	if err != nil {
		return fmt.Errorf("Cannot lock `%s'.", o.confPath())
	}
	// unlock will be done at deffered f.Close()

	dec := json.NewDecoder(f)
	if err = dec.Decode(o); err != nil {
		return fmt.Errorf("Config file parsing error: %s", err)
	}

	return nil
}

func (o *Options) Save() error {
	if err := os.MkdirAll(path.Dir(o.confPath()), 0755); err != nil {
		return fmt.Errorf("Could not create directory for file `%s':  %s", o.confPath(), err)
	}

	f, err := os.OpenFile(o.confPath(), os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("Could not open config file `%s':  %s", o.confPath(), err)
	}
	defer f.Close()

	err = syscall.Flock(int(f.Fd()), C.LOCK_EX)
	if err != nil {
		return fmt.Errorf("Cannot lock `%s'.", o.confPath())
	}

	f.Seek(0, 0)
	f.Truncate(0)

	data, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return fmt.Errorf("Could not jsonify options: %s", err)
	}

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("Could not save data to `%s' :   %s", o.confPath(), err)
	}

	if _, err := f.WriteString("\n"); err != nil {
		return fmt.Errorf("Could not save data to `%s' :   %s", o.confPath(), err)
	}

	return nil
}

var options = &Options{}

var parser = flags.NewParser(options, flags.Default)

func init() {
	options.BaseFlag = func(arg string) error {
		options.BaseDir = arg
		options.Load()
		return nil
	}
}
