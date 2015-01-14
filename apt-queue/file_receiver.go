package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/fsnotify.v1"
)

type Packagereceiver interface {
	Next() (string, error)
	Release(string)
}

type NotifyFileReceiver struct {
	watcheddir   string
	stagedir     string
	staggedfiles map[string][]string

	watcher   *fsnotify.Watcher
	files     chan string
	torelease chan string
	errors    chan error
}

func NewNotifyFileReceiver(dir string) (*NotifyFileReceiver, error) {
	res := &NotifyFileReceiver{
		watcheddir:   dir,
		files:        make(chan string, 1),
		torelease:    make(chan string),
		errors:       make(chan error),
		staggedfiles: make(map[string][]string),
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	res.stagedir = path.Join(path.Dir(abs), "incoming-stagged")

	toEnsureEmpty := []string{dir, res.stagedir}
	for _, d := range toEnsureEmpty {
		if err = os.RemoveAll(d); err != nil {
			return nil, err
		}
		if err = os.MkdirAll(d, 0755); err != nil {
			return nil, err
		}
	}

	err = os.MkdirAll(res.stagedir, 0755)
	if err != nil {
		return nil, err
	}

	res.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	err = res.watcher.Add(dir)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case f := <-res.torelease:
				if err := res.release(f); err != nil {
					res.errors <- err
				}
			case ev := <-res.watcher.Events:
				if err := res.handleEvent(ev); err != nil {
					res.errors <- err
				}
			case err := <-res.watcher.Errors:
				res.errors <- err
			}
		}
	}()

	return res, err

}

func (r *NotifyFileReceiver) handleEvent(ev fsnotify.Event) error {
	if ev.Op&fsnotify.Chmod == 0 {
		return nil
	}

	ext := path.Ext(ev.Name)
	switch ext {
	case ".changes", ".deb", ".udeb", ".dsc":
		break
	case ".gz":
		switch {
		case strings.HasSuffix(ev.Name, ".orig.tar.gz"):
			break
		case strings.HasSuffix(ev.Name, ".diff.gz"):
			break
		case strings.HasSuffix(ev.Name, ".tar.gz"):
			break
		default:
			return nil
		}
	default:
		return nil
	}
	basename := path.Base(ev.Name)
	_, exists := r.staggedfiles[basename]
	if exists == false {
		dest := path.Join(r.stagedir, basename)
		if err := os.Rename(ev.Name, dest); err != nil {
			return err
		}
		r.staggedfiles[basename] = []string{}
		go func() {
			r.files <- dest
		}()
		return nil
	}

	destfile, err := ioutil.TempFile(r.stagedir, basename+".")
	if err != nil {
		return err
	}

	dest := destfile.Name()
	destfile.Close()

	err = os.Rename(ev.Name, dest)

	r.staggedfiles[basename] = append(r.staggedfiles[basename], dest)
	return nil
}

func (r *NotifyFileReceiver) Next() (string, error) {
	select {
	case err := <-r.errors:
		return "", err
	case f := <-r.files:
		return f, nil
	}
}

func (r *NotifyFileReceiver) release(pathname string) error {
	if path.Dir(pathname) != r.stagedir {
		return fmt.Errorf("file %s is not in %s", pathname, r.stagedir)
	}
	basename := path.Base(pathname)
	_, ok := r.staggedfiles[basename]
	if ok == false {
		return fmt.Errorf("Could not release unstored file %s", pathname)
	}
	err := os.Remove(pathname)
	if err != nil {
		return err
	}

	if len(r.staggedfiles[basename]) == 0 {
		delete(r.staggedfiles, basename)
		return nil
	}

	newfile := r.staggedfiles[basename][0]
	err = os.Rename(newfile, path.Join(r.stagedir, basename))
	if err != nil {
		return err
	}
	r.staggedfiles[basename] = r.staggedfiles[basename][1:]

	// signal a new one is here
	go func() {
		r.files <- pathname
	}()

	return nil
}

func (r *NotifyFileReceiver) Release(path string) {
	go func() {
		r.torelease <- path
	}()
}
