package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"sort"

	deb ".."
	"github.com/nightlyone/lockfile"
	"launchpad.net/go-xdg"
)

type XdgUserDistConfig struct {
	supported map[deb.Codename]map[deb.Architecture]bool

	dataPath string
	lock     lockfile.Lockfile
}

func NewXdgUserDistConfig() (*XdgUserDistConfig, error) {
	res := &XdgUserDistConfig{
		supported: make(map[deb.Codename]map[deb.Architecture]bool),
	}
	var err error
	res.dataPath, err = xdg.Config.Ensure("go-deb.builder/dist-config.json")
	if err != nil {
		return nil, err
	}

	res.lock, err = lockfile.New(path.Join(path.Dir(res.dataPath), "dist-config.lock"))
	if err != nil {
		return nil, err
	}

	err = res.load()
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *XdgUserDistConfig) tryLock() error {
	if err := c.lock.TryLock(); err != nil {
		return fmt.Errorf("Could not lock %s: %s", c.lock, err)
	}
	return nil
}

func (c *XdgUserDistConfig) unlockOrPanic() {
	if err := c.lock.Unlock(); err != nil {
		panic(err)
	}
}

func (c *XdgUserDistConfig) load() error {
	if err := c.tryLock(); err != nil {
		return err
	}
	defer c.unlockOrPanic()

	f, err := os.Open(c.dataPath)
	if err != nil {
		return err
	}

	dec := json.NewDecoder(f)
	err = dec.Decode(&c.supported)
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}

func (c *XdgUserDistConfig) save() error {
	if err := c.tryLock(); err != nil {
		return err
	}
	defer c.unlockOrPanic()

	f, err := os.Create(c.dataPath)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(f)
	return enc.Encode(c.supported)
}

func (c *XdgUserDistConfig) Supported() map[deb.Codename]ArchitectureList {
	res := make(map[deb.Codename]ArchitectureList)
	for d, archs := range c.supported {
		list := make(ArchitectureList, 0, len(archs))
		for a, _ := range archs {
			list = append(list, a)
		}
		sort.Sort(list)
		res[d] = list
	}
	return res
}

func (c *XdgUserDistConfig) Add(d deb.Codename, a deb.Architecture) error {
	oldSupp, ok := c.supported[d]
	if ok == false {
		c.supported[d] = make(map[deb.Architecture]bool)
	}
	c.supported[d][a] = true
	if err := c.save(); err != nil {
		if ok == false {
			delete(c.supported, d)
		} else {
			c.supported[d] = oldSupp
		}
		return err
	}
	return nil
}

func (c *XdgUserDistConfig) Remove(d deb.Codename, a deb.Architecture) error {
	oldArchs, ok := c.supported[d]
	delete(c.supported[d], a)
	if len(c.supported[d]) == 0 {
		delete(c.supported, d)
	}

	if err := c.save(); err != nil {
		// condition always true here, len(oldArchs) >= 1 (contains at
		// least 'a')
		if ok == true {
			c.supported[d] = oldArchs
		}

		return err
	}

	return nil
}
