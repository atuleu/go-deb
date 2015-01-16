package main

import (
	"fmt"
	"log"
	"net/mail"
	"os"
	"path"
	"strings"
)

type ListenCommand struct {
	Dir string `short:"D" long:"dir" description:"Directory to listen to"`

	DefaultMail string `short:"m" long:"mail" description:"default mail to send failure to"`

	errorMail *mail.Address

	fileReceiver    *NotifyFileReceiver
	openedReference map[string]*QueueFileReference
}

func (x *ListenCommand) handleChanges(ref *QueueFileReference) error {

	i, err := NewInteractor(options)
	if err != nil {
		return err
	}

	res, err := i.ProcessChangesFile(ref, nil)
	defer func() {
		for _, f := range res.FilesToRemove {
			_, ok := x.openedReference[f.Id()]
			if ok == true {
				x.fileReceiver.Release(f)
			}
		}
		x.fileReceiver.Release(ref)
	}()

	if err == nil {
		//TODO: send a mail to use that it is

		log.Printf("Included %s", ref.Id())

		return nil
	}

	//TODO: prepare a failure mail

	if res.ShouldReport == true {
		//TODO: send an email to both admin and user if there is one

	} else {
		//TODO: send an email only to maintainer
	}

	log.Printf("Could not include %s: %s", ref.Id(), err)

	return nil
}

func (x *ListenCommand) Execute(args []string) error {
	x.openedReference = make(map[string]*QueueFileReference)
	defer func() {
		for _, ref := range x.openedReference {
			x.fileReceiver.Release(ref)
		}
	}()

	if len(args) != 0 {
		return fmt.Errorf("Takes no argument")
	}
	var err error
	x.errorMail, err = mail.ParseAddress(x.DefaultMail)
	if err != nil {
		log.Printf("[WARNING]: Could not set mail to %s: %s", x.DefaultMail, err)
		x.errorMail = nil
	}

	x.fileReceiver, err = NewNotifyFileReceiver(x.Dir)
	if err != nil {
		return err
	}
	log.Printf("Watching event in %s", x.Dir)

	for {

		ref, err := x.fileReceiver.Next()
		if err != nil {
			return err
		}

		if strings.HasSuffix(ref.Name, ".changes") == false {
			x.openedReference[ref.Id()] = ref
			continue
		}

		if err = x.handleChanges(ref); err != nil {
			return err
		}
	}

}

func init() {
	parser.AddCommand("listen",
		"Listen for incoming .changes file",
		"Listen for incoming .changes file",
		&ListenCommand{
			Dir:         path.Join(os.Getenv("HOME"), "incoming"),
			DefaultMail: os.Getenv("USER") + "@localhost",
		})
}
