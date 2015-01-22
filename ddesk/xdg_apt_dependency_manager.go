package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/nightlyone/lockfile"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"launchpad.net/go-xdg"
)

type XdgAptDependencyManager struct {
	lock     lockfile.Lockfile
	confdir  string
	confpath string

	data map[AptRepositoryId]*AptRepositoryAccess
}

func NewXdgAptDependencyManager() (*XdgAptDependencyManager, error) {
	res := &XdgAptDependencyManager{
		data: make(map[AptRepositoryId]*AptRepositoryAccess),
	}
	var err error
	res.confpath, err = xdg.Data.Ensure("go-deb.ddesk/apt_dependencies/data.json")
	if err != nil {
		return nil, err
	}
	res.confdir = path.Dir(res.confpath)
	res.lock, err = lockfile.New(path.Join(res.confdir, "global.lock"))
	if err != nil {
		return nil, err
	}

	err = res.load()
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *XdgAptDependencyManager) tryLock() error {
	if err := m.lock.TryLock(); err != nil {
		return fmt.Errorf("Could not lock %s: %s", m.lock, err)
	}
	return nil
}

func (m *XdgAptDependencyManager) unlock() {
	if err := m.lock.Unlock(); err != nil {
		panic(err)
	}
}

func (m *XdgAptDependencyManager) keyPath(id AptRepositoryId) string {
	return ""
}

func (m *XdgAptDependencyManager) load() error {
	if err := m.tryLock(); err != nil {
		return err
	}
	defer m.unlock()

	f, err := os.Open(m.confpath)
	if err != nil && os.IsNotExist(err) == false {
		return err
	}
	dec := json.NewDecoder(f)

	err = dec.Decode(m.data)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}

	for id, _ := range m.data {
		m.data[id].PublicKey, err = m.loadKey(id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *XdgAptDependencyManager) loadKey(id AptRepositoryId) (*openpgp.Entity, error) {
	f, err := os.Open(m.keyPath(id))
	if err != nil {
		return nil, err
	}

	keyring, err := openpgp.ReadArmoredKeyRing(f)
	if err != nil {
		return nil, err
	}
	if len(keyring) != 1 {
		return nil, fmt.Errorf("Wrong number of key (got: %d) in %s", len(keyring), m.keyPath(id))
	}

	return keyring[0], nil
}

func (m *XdgAptDependencyManager) save() error {
	if err := m.tryLock(); err != nil {
		return err
	}
	defer m.unlock()

	f, err := os.Create(m.confpath)
	if err != nil {
		return err
	}
	defer f.Close()

	for id, _ := range m.data {
		err = m.saveKey(id)
		if err != nil {
			return err
		}
	}

	enc := json.NewEncoder(f)
	err = enc.Encode(m.data)
	if err != nil {
		return err
	}

	return nil
}

func (m *XdgAptDependencyManager) saveKey(id AptRepositoryId) error {
	access, ok := m.data[id]
	if ok == false {
		return fmt.Errorf("Unknown repository %s", id)
	}
	if access.PublicKey == nil {
		return fmt.Errorf("No key found for %s", id)
	}
	f, err := os.Create(m.keyPath(id))
	if err != nil {
		return err
	}
	defer f.Close()

	w, err := armor.Encode(f, "PGP PUBLIC KEY BLOCK", nil)
	if err != nil {
		return err
	}
	defer w.Close()

	err = access.PublicKey.Serialize(w)
	if err != nil {
		w.Close()
		f.Close()
		os.Remove(f.Name())
		return err
	}

	return nil
}

func (m *XdgAptDependencyManager) Store(a *AptRepositoryAccess) error {
	m.data[a.ID] = a
	return m.save()
}

func (m *XdgAptDependencyManager) Remove(id AptRepositoryId) error {
	if _, ok := m.data[id]; ok == false {
		return fmt.Errorf("%s repository is not listed", id)
	}

	err := os.Remove(m.keyPath(id))
	if err != nil {
		return err
	}

	delete(m.data, id)

	return m.save()
}

func (m *XdgAptDependencyManager) List() map[AptRepositoryId]*AptRepositoryAccess {
	return m.data
}
