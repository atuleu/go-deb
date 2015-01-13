package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"gopkg.in/fsnotify.v1"

	deb ".."
)

type ChangeFileListener struct {
	dir     string
	watcher *fsnotify.Watcher
	logger  *log.Logger

	filesRemover chan string
}

func NewChangeFileListener(d string) (*ChangeFileListener, error) {
	res := &ChangeFileListener{
		dir:          d,
		logger:       log.New(os.Stderr, "", log.LstdFlags),
		filesRemover: make(chan string, 0),
	}
	var err error
	res.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (l *ChangeFileListener) handleEvent(e fsnotify.Event) {
	if e.Op&(fsnotify.Create|fsnotify.Remove) == 0 {
		l.logger.Printf("Ignoring event: %s", e)
		return
	}

	if e.Op&fsnotify.Remove != 0 {
		l.logger.Printf("File cleaning not yet implemented")
		return
	}

	//we have created a file.

	switch path.Ext(e.Name) {
	case ".changes":
		l.logger.Printf(".changes file are not yet implemented")
		return
	case ".deb", ".dsc", ".udeb":
		l.logger.Printf("debian file are not yet implemented")
		return
	case ".gz":
		if strings.HasSuffix(e.Name, ".orig.tar.gz") || strings.HasSuffix(e.Name, ".diff.gz") {
			l.logger.Printf("original tarball or diff not handled yet")
			return
		}
	}

	err := os.Remove(e.Name)
	if err != nil {
		l.logger.Fatalf("Could not remove unhandled file %s: %s", e.Name, err)
	}
}

func (l *ChangeFileListener) RemoveFile(f string) error {
	return deb.NotYetImplemented()
}

func (l *ChangeFileListener) Listen() error {
	err := l.watcher.Add(l.dir)
	if err != nil {
		return fmt.Errorf("Could not watch directory %s: %s", l.dir, err)
	}
	l.logger.Printf("Waiting for .changes file in %s", l.dir)
	for {
		select {
		case event := <-l.watcher.Events:
			l.handleEvent(event)
		case err := <-l.watcher.Errors:
			if err != nil {
				l.logger.Fatalf("Got watcher error: %s", err)
			}
		case f := <-l.filesRemover:
			if err := l.RemoveFile(f); err != nil {
				l.logger.Fatalf("Could not remove file: %s", err)
			}
		}
	}

}

type ListenCommand struct {
	Dir string `short:"D" long:"dir" description:"directory to watch file"`
}

func (x *ListenCommand) Execute(args []string) error {
	l, err := NewChangeFileListener(x.Dir)
	if err != nil {
		return err
	}

	return l.Listen()

}

func init() {
	parser.AddCommand("listen",
		"Listen for incoming .changes file in a directory",
		"Listen for incoming .changes file in a directory",
		&ListenCommand{
			Dir: path.Join(os.Getenv("HOME"), "incoming"),
		})
}
