package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/fsnotify.v1"

	deb ".."
)

type QueueFileReference struct {
	Name      string
	Component deb.Component
	dir       string
}

func (r *QueueFileReference) Id() string {
	if len(r.Component) != 0 {
		return path.Join(string(r.Component), r.Name)
	}
	return r.Name
}

func (r *QueueFileReference) Path() string {
	return path.Join(r.dir, string(r.Component), r.Name)
}

type PackageReceiver interface {
	Next() (*QueueFileReference, error)
	Release(*QueueFileReference)
}

type NotifyFileReceiver struct {
	watcheddir  string
	stagedir    string
	stagedFiles map[string][]string

	watcher   *fsnotify.Watcher
	files     chan *QueueFileReference
	torelease chan *QueueFileReference
	errors    chan error
}

func NewNotifyFileReceiver(dir string, username string) (*NotifyFileReceiver, error) {
	res := &NotifyFileReceiver{
		watcheddir:  dir,
		files:       make(chan *QueueFileReference, 1),
		torelease:   make(chan *QueueFileReference),
		errors:      make(chan error),
		stagedFiles: make(map[string][]string),
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	res.stagedir = path.Join(path.Dir(abs), "incoming-staging")

	var watchingUser *user.User = nil
	if len(username) != 0 {
		watchingUser, err = user.Lookup(username)
		if err != nil {
			return nil, err
		}
	}

	toEnsureEmpty := []string{dir, res.stagedir}
	for _, d := range toEnsureEmpty {
		if err = os.RemoveAll(d); err != nil {
			return nil, err
		}
		if err = os.MkdirAll(d, 0755); err != nil {
			return nil, err
		}

		if watchingUser == nil {
			continue
		}

		uid, err := strconv.ParseInt(watchingUser.Uid, 0, 32)
		if err != nil {
			return nil, err
		}
		gid, err := strconv.ParseInt(watchingUser.Gid, 0, 32)
		if err != nil {
			return nil, err
		}

		if err = os.Chown(d, int(uid), int(gid)); err != nil {
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
	if ev.Op == fsnotify.Create {
		info, err := os.Stat(ev.Name)
		if err != nil {
			return err
		}
		if info.IsDir() == true {
			return r.watcher.Add(ev.Name)
		}
	}

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
	case ".xz":
		switch {
		case strings.HasSuffix(ev.Name, ".orig.tar.xz"):
			break
		case strings.HasSuffix(ev.Name, ".tar.xz"):
			break
		default:
			return nil
		}
	default:
		return nil
	}

	fDir := path.Dir(ev.Name)
	component := ""
	if fDir != r.watcheddir {
		component = path.Base(fDir)
		fDir = path.Dir(fDir)
		if fDir != r.watcheddir {
			log.Printf("File %s is not in directory %s(/<component>)?/", ev.Name, r.watcheddir)
			return nil
		}
		err := os.MkdirAll(path.Join(r.stagedir, component), 0755)
		if err != nil {
			return err
		}
	}
	ref := &QueueFileReference{
		Name:      path.Base(ev.Name),
		Component: deb.Component(component),
		dir:       r.stagedir,
	}
	_, exists := r.stagedFiles[ref.Id()]
	if exists == false {
		if err := os.Rename(ev.Name, ref.Path()); err != nil {
			return err
		}
		r.stagedFiles[ref.Id()] = []string{}
		go func() {
			r.files <- ref
		}()
		return nil
	}

	destfile, err := ioutil.TempFile(path.Join(r.stagedir, string(ref.Component)), ref.Name+".")
	if err != nil {
		return err
	}

	dest := destfile.Name()
	destfile.Close()

	err = os.Rename(ev.Name, dest)

	r.stagedFiles[ref.Id()] = append(r.stagedFiles[ref.Id()], dest)
	return nil
}

func (r *NotifyFileReceiver) Next() (*QueueFileReference, error) {
	select {
	case err := <-r.errors:
		return nil, err
	case f := <-r.files:
		log.Printf("Received %s", f.Id())
		return f, nil
	}
}

func (r *NotifyFileReceiver) release(ref *QueueFileReference) error {
	if ref.dir != r.stagedir {
		return fmt.Errorf("file %s is not in %s", ref.Id(), r.stagedir)
	}

	_, ok := r.stagedFiles[ref.Id()]
	if ok == false {
		return fmt.Errorf("Could not release unstored file %s", ref.Id())
	}
	err := os.Remove(ref.Path())
	if err != nil {
		return err
	}

	if len(r.stagedFiles[ref.Id()]) == 0 {
		delete(r.stagedFiles, ref.Id())
		return nil
	}

	newfile := r.stagedFiles[ref.Id()][0]
	err = os.Rename(newfile, ref.Path())
	if err != nil {
		return err
	}
	r.stagedFiles[ref.Id()] = r.stagedFiles[ref.Id()][1:]

	// signal a new one is here
	go func() {
		r.files <- ref
	}()

	return nil
}

func (r *NotifyFileReceiver) Release(ref *QueueFileReference) {
	go func() {
		log.Printf("deleting %s", ref.Id())
		r.torelease <- ref
	}()
}
