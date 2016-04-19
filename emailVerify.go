package main

import (
	"bytes"
	log "github.com/Sirupsen/logrus"
	"net/smtp"
	"os"
	"text/template"
)

type emailer struct {
	Server   *emailServer
	Template smtpTemplate
	Auth     smtp.Auth
	EmailDoc bytes.Buffer
}

type emailServer struct {
	Username string
	Password string
	Address  string
	Port     string
}

type smtpTemplate struct {
	From      string
	To        string
	Subject   string
	VerifyURL string
}

const emailTemplate = `From: {{.From}}
To: {{.To}}}
Subject: {{.Subject}}
Welcome to Social Hardware!

Verify yourself and start logging data.

To get started, click the link below:
{{.VerifyURL}}

Sincerely,
{{.From}}
`

//////////////////////////////////////////////////////////////////////////
//
//
//
//
//////////////////////////////////////////////////////////////////////////
func NewEmailer() *emailer {
	return &emailer{}
}

//////////////////////////////////////////////////////////////////////////
//
//
//
//
//////////////////////////////////////////////////////////////////////////
func (mail *emailer) Send(to string, token string, url string) {
	mail.connect()
	mail.create(to, token, url)
	mail.deliver(to)
}

//////////////////////////////////////////////////////////////////////////
//
//
//
//
//////////////////////////////////////////////////////////////////////////
func (mail *emailer) connect() {

	username := os.Getenv("SOCIALHW_EMAIL_USERNAME")
	pw := os.Getenv("SOCIALHW_EMAIL_PW")
	url := os.Getenv("SOCIALHW_EMAIL_URL")
	port := os.Getenv("SOCIALHW_EMAIL_PORT")

	server := &emailServer{username, pw, url, port}
	mail.Server = server
	mail.Auth = smtp.PlainAuth("", server.Username, server.Password, server.Address)

}

//////////////////////////////////////////////////////////////////////////
//
//
//
//
//////////////////////////////////////////////////////////////////////////
func (mail *emailer) create(to string, token string, url string) {

	context := &smtpTemplate{
		"Social Hardware",
		to,
		"Email Verification",
		url,
	}

	//create the template from the string. Could load from a file?
	t := template.New("emailTemplate")

	//parse the template
	t, err := t.Parse(emailTemplate)
	if err != nil {
		log.Errorf("emailTemplate error %s", err)
	}

	err = t.Execute(&mail.EmailDoc, context)
	if err != nil {
		log.Errorf("template exec error %s", err)
	}
}

//////////////////////////////////////////////////////////////////////////
//
//
//
//
//////////////////////////////////////////////////////////////////////////
func (mail *emailer) deliver(userEmail string) {

	addressPort := mail.Server.Address + ":" + mail.Server.Port

	err := smtp.SendMail(addressPort, mail.Auth, mail.Server.Username, []string{userEmail}, mail.EmailDoc.Bytes())
	if err != nil {
		log.Errorf("sendmail error %s", err)
	}
}
