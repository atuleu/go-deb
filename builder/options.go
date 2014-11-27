package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	"launchpad.net/go-xdg"
)

// import "launchpad.net/go-xdg"

type Options struct {
}

var configSuffix = "go-deb.builder/config.json"

func ensureConfigFile() (string, error) {
	confPath, err := xdg.Config.Find(configSuffix)
	if err != nil {
		confPath, err = xdg.Config.Ensure(configSuffix)
		if err != nil {
			return "", fmt.Errorf("Could not create user config : %s", confPath)
		}
	}
	return confPath, nil
}

func (o *Options) LoadFromXDG() error {
	confPath, err := ensureConfigFile()
	if err != nil {
		return err
	}
	f, err := os.Open(confPath)
	if err != nil {
		return fmt.Errorf("Could not open config file `%s' : %s", confPath, err)
	}

	defer f.Close()

	dec := json.NewDecoder(f)
	if err = dec.Decode(o); err != nil {
		return fmt.Errorf("Config file parsing error: %s", err)
	}

	return nil
}

func (o *Options) SaveToXDG() error {
	confPath, err := ensureConfigFile()
	if err != nil {
		return err
	}

	f, err := os.Create(confPath)
	if err != nil {
		return fmt.Errorf("Could not create config file `%s':  %s", confPath, err)
	}

	defer f.Close()

	dec := json.NewEncoder(f)
	if err = dec.Encode(o); err != nil {
		return fmt.Errorf("Could not save config to `%s': %s", confPath, err)
	}

	return nil
}

var options = &Options{}

var parser = flags.NewParser(options, flags.Default)
