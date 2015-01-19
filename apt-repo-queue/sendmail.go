package main

import (
	"fmt"
	"net/mail"
	"os/exec"
	"strings"
)

type SendMail struct {
	From mail.Address
}

func NewSendMail(name, addr string) *SendMail {
	return &SendMail{
		From: mail.Address{
			Name:    name,
			Address: addr,
		},
	}
}

func (x *SendMail) SendMail(to []*mail.Address, subject, corpus string) error {
	var justMail, fullAddress []string
	for _, e := range to {
		justMail = append(justMail, e.Address)
		fullAddress = append(fullAddress, e.String())
	}

	header := make(map[string]string)
	header["From"] = x.From.String()
	header["To"] = strings.Join(fullAddress, ", ")
	header["Subject"] = strings.Trim(subject, " <>\n")
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = `text/html; charset="UTF-8"`
	//header["Content-Transfer-Encoding"] = "base64"

	dataHeader := ""
	for k, v := range header {
		dataHeader += fmt.Sprintf("%s: %s\n", k, v)
	}

	inBuffer := strings.NewReader(dataHeader + "\n" + corpus)
	cmd := exec.Command("sendmail", "-t", strings.Join(justMail, " "))
	cmd.Stdin = inBuffer

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Could not sendmail: %s", err)
	}

	return nil
}
