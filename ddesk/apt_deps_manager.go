package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/nightlyone/lockfile"
	"launchpad.net/go-xdg"
)

// AptDepsManager is managing apt repository dependency in a
// persistent manner.
type AptDepsManager interface {
	Store(*AptRepositoryAccess) error
	Remove(AptRepositoryID) error
	List() map[AptRepositoryID]*AptRepositoryAccess
}

// XdgAptDepsManager implements AptDepsManager by storing the
// dependencies in the current user XDG_DATA_DIR
type XdgAptDepsManager struct {
	lock              lockfile.Lockfile
	confdir, confpath string

	data map[AptRepositoryID]*AptRepositoryAccess
}

// NewXdgAptDepsManager restores an AptDepsManager from an user
// XDG_DATA_DIR
func NewXdgAptDepsManager() (*XdgAptDepsManager, error) {
	res := &XdgAptDepsManager{
		data: make(map[AptRepositoryID]*AptRepositoryAccess),
	}
	var err error
	res.confpath, err = xdg.Data.Ensure("go-deb.ddesk/apt_deps/data.json")
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

func (m *XdgAptDepsManager) tryLock() error {
	if err := m.lock.TryLock(); err != nil {
		return fmt.Errorf("Could not lock %s: %s", m.lock, err)
	}
	return nil
}

func (m *XdgAptDepsManager) unlock() {
	if err := m.lock.Unlock(); err != nil {
		panic(err)
	}
}

func (m *XdgAptDepsManager) load() error {
	if err := m.tryLock(); err != nil {
		return err
	}
	defer m.unlock()

	f, err := os.Open(m.confpath)
	if err != nil {
		if os.IsNotExist(err) {
			return err
		}
		return err
	}
	dec := json.NewDecoder(f)
	err = dec.Decode(&m.data)
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

func (m *XdgAptDepsManager) save() error {
	if err := m.tryLock(); err != nil {
		return err
	}
	defer m.unlock()

	f, err := os.Create(m.confpath)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	err = enc.Encode(m.data)
	if err != nil {
		return err
	}

	return nil
}

// Store is saving to the filesystem the given AptRepositoryAccess. It
// returns an error on any failure. The actual state of the object
// will not be modified on failure.
func (m *XdgAptDepsManager) Store(a *AptRepositoryAccess) error {
	if a == nil {
		return nil
	}
	saved, ok := m.data[a.ID]
	m.data[a.ID] = a

	err := m.save()
	if err != nil {
		if ok == false {
			delete(m.data, a.ID)
		} else {
			m.data[a.ID] = saved
		}
	}
	return err
}

// Remove is removing from the persistent state the given
// AptRepositoryAccess. If it returns any error the in memory state
// will be left unchanged.
func (m *XdgAptDepsManager) Remove(id AptRepositoryID) error {
	saved, ok := m.data[id]
	if ok == false {
		return fmt.Errorf("%s is not listed", id)
	}

	delete(m.data, id)
	err := m.save()
	if err != nil {
		m.data[id] = saved
	}
	return err
}

// List returns the current in-memory state of the list of apt
// repository dependencies.
func (m *XdgAptDepsManager) List() map[AptRepositoryID]*AptRepositoryAccess {
	return m.data
}
