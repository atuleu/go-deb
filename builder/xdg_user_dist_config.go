package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"

	deb ".."
	"github.com/nightlyone/lockfile"
	"launchpad.net/go-xdg"
)

type XdgUserDistConfig struct {
	supported map[deb.Distribution][]deb.Architecture

	dataPath string
	lock     lockfile.Lockfile
}

func NewXdgUserDistConfig() (*XdgUserDistConfig, error) {
	res := &XdgUserDistConfig{
		supported: make(map[deb.Distribution][]deb.Architecture),
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

func (c *XdgUserDistConfig) Supported() map[deb.Distribution][]deb.Architecture {
	return c.supported
}

func (c *XdgUserDistConfig) Add(d deb.Distribution, a deb.Architecture) error {
	oldSupp := c.supported[d]
	c.supported[d] = append(oldSupp, a)
	if err := c.save(); err != nil {
		if len(oldSupp) == 0 {
			delete(c.supported, d)
		} else {
			c.supported[d] = oldSupp
		}
		return err
	}
	return nil
}

func (c *XdgUserDistConfig) Remove(d deb.Distribution, a deb.Architecture) error {
	oldArchs := c.supported[d]
	newArchs := []deb.Architecture{}
	found := false

	for _, aa := range oldArchs {
		if a == aa {
			found = true
			continue
		}
		newArchs = append(newArchs, a)
	}

	if found == false {
		return fmt.Errorf("distribution %s does not supports %s", d, a)
	}

	if len(newArchs) == 0 {
		delete(c.supported, d)
	} else {
		c.supported[d] = newArchs
	}

	if err := c.save(); err != nil {
		// condition always true here, len(oldArchs) >= 1 (contains at
		// least 'a')
		c.supported[d] = oldArchs

		return err
	}

	return nil
}
