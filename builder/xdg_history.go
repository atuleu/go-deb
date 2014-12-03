package main

import (
	"encoding/json"
	"os"
	"path"

	"github.com/nightlyone/lockfile"
	"launchpad.net/go-xdg"

	deb ".."
)

// An history made with a locked .json object in the XDG_DATA_HOME of
// the user
type XdgHistory struct {
	filepath string
	lock     lockfile.Lockfile

	data []deb.SourcePackageRef
}

var xdgHistoryPath = "go-deb.builder/history/data.json"

func NewXdgHistory() (*XdgHistory, error) {
	res := &XdgHistory{}
	var err error
	res.filepath, err = xdg.Data.Ensure(xdgHistoryPath)
	if err != nil {
		return nil, err
	}

	lockPath := path.Join(path.Dir(res.filepath), "data.lock")

	res.lock, err = lockfile.New(lockPath)
	if err != nil {
		return nil, err
	}

	err = res.load()
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (h *XdgHistory) load() error {
	if err := h.lock.TryLock(); err != nil {
		return err
	}
	defer func() {
		if err := h.lock.Unlock(); err != nil {
			panic(err)
		}
	}()

	f, err := os.Open(h.filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	return dec.Decode(h.data)
}

func (h *XdgHistory) save() error {
	if err := h.lock.TryLock(); err != nil {
		return err
	}
	defer func() {
		if err := h.lock.Unlock(); err != nil {
			panic(err)
		}
	}()

	f, err := os.Create(h.filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	return enc.Encode(h.data)
}

func (h *XdgHistory) Append(p deb.SourcePackageRef) {
	toSave := h.data
	if len(h.data) >= 20 {
		toSave = h.data[0:19]
	}
	h.data = append([]deb.SourcePackageRef{p}, toSave...)
	if err := h.save(); err != nil {
		panic(err)
	}
}
func (h *XdgHistory) Get() []deb.SourcePackageRef {
	return h.data
}

func (h *XdgHistory) RemoveFront(p deb.SourcePackageRef) {
	i := 0
	for i = 0; i < len(h.data); i = i + 1 {
		if h.data[i] != p {
			break
		}
	}

	if i == 0 {
		return
	}

	h.data = h.data[i:]
	if err := h.save(); err != nil {
		panic(err)
	}
}
