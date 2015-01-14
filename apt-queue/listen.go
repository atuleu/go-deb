package main

import (
	"fmt"
	"log"
	"net/mail"
	"os"
	"path"
	"strings"
	"time"
)

type ListenCommand struct {
	Dir string `short:"D" long:"dir" description:"Directory to listen to"`

	DefaultMail string `short:"m" long:"mail" description:"default mail to send failure to"`

	errorMail *mail.Address

	r *NotifyFileReceiver
}

func (x *ListenCommand) handleChanges(pathname string) error {
	x, err := NewInteractor(options)
	if err != nil {
		return err
	}

	res, err := x.ProcessChangesFile(pathname)

	if err != nil {
		//TODO: send a mail to use that it is sucessful
		log.Printf("Included %s", pathname)

		return nil
	}

	//TODO: prepare a failure mail

	if res.ShouldReport == true {
		//TODO: send an email to both admin and user if there is one

	} else {
		//TODO: send an email only to maintainer
	}

}

func (x *ListenCommand) Execute(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("Takes no argument")
	}
	var err error
	x.errorMail, err = mail.ParseAddress(x.DefaultMail)
	if err != nil {
		return err
	}

	r, err := NewNotifyFileReceiver(x.Dir)
	if err != nil {
		return err
	}
	log.Printf("Watching event in %s", x.Dir)

	var nextDate *time.Time = nil
	for {

		f, err := r.Next()
		if err != nil {
			return err
		}

		if strings.HasSuffix(".changes") == false {
			continue
		}

		if err = x.handleChanges(f); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	parser.AddCommand("listen",
		"Listen for incoming .changes file",
		"Listen for incoming .changes file",
		&ListenCommand{
			Dir:         path.Join(os.Getenv("HOME"), "incoming"),
			DefaultMail: os.Getenv("USER"),
		})
}
