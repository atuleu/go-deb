package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/mail"
	"os"
	"path"
	"strings"
)

type TemplateArgs struct {
	AllComp, Succeed                 bool
	Error, Comp, ChangesName, Output string
}

type ListenCommand struct {
	Dir string `short:"D" long:"dir" description:"Directory to listen to"`

	errorMail *mail.Address

	fileReceiver    *NotifyFileReceiver
	openedReference map[string]*QueueFileReference
	mailer          *SendMail
}

func (x *ListenCommand) handleChanges(ref *QueueFileReference) error {
	mailTemplate, err := template.New("mail").Parse(`<p> This mail is automatically sent by go-deb.apt-queue </p> 
<h2> The inclusion of {{.ChangesName}} in {{if .AllComp }} all component {{else}} {{.Comp}} {{end}} {{if .Succeed}} succeed {{else}} failed {{end}}:</h2>
{{if .Succeed }} {{else}}<p> Error is : {{.Error}} </p> {{end}}
<h3>Reprepro output: </h3>
<pre>{{.Output}}</pre>
`)

	if err != nil {
		return err
	}

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

	if res.ShouldReport == true {
		res.SendTo = append(res.SendTo, x.errorMail)
	}
	var subject string
	messageArgs := TemplateArgs{
		Output:      string(res.Output),
		ChangesName: ref.Name,
		Comp:        string(ref.Component),
		AllComp:     len(ref.Component) == 0,
		Succeed:     err == nil,
	}

	if err == nil {
		subject = fmt.Sprintf("Inclusion of %s succeed", ref.Id())
		log.Printf("Included %s", ref.Id())
	} else {
		subject = fmt.Sprintf("Inclusion of %s failed", ref.Id())
		log.Printf("Could not include %s: %s", ref.Id(), err)
		messageArgs.Error = fmt.Sprintf("%s", err)
	}
	var message bytes.Buffer

	if err = mailTemplate.Execute(&message, messageArgs); err != nil {
		return err
	}

	if err = x.mailer.SendMail(res.SendTo, subject, message.String()); err != nil {
		return err
	}

	return nil
}

func (x *ListenCommand) Execute(args []string) error {
	config, err := LoadConfig(options.Base)
	if err != nil {
		return err
	}

	x.mailer = NewSendMail(config.KeyName, config.KeyEmail)

	x.openedReference = make(map[string]*QueueFileReference)
	defer func() {
		for _, ref := range x.openedReference {
			x.fileReceiver.Release(ref)
		}
	}()

	if len(args) != 0 {
		return fmt.Errorf("Takes no argument")
	}
	x.errorMail, err = mail.ParseAddress(config.KeyEmail)
	if err != nil {
		log.Printf("[WARNING]: Could not set mail to %s: %s", config.KeyEmail, err)
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
			Dir: path.Join(os.Getenv("HOME"), "incoming"),
		})
}
