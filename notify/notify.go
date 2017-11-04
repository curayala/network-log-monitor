package notify

import (
	"bytes"
	"html/template"

	gomail "gopkg.in/gomail.v2"
)

var emailContentFile, _ = Asset("templates/email-content.template")
var emailContentTemplate = template.Must(template.New("email-content").Parse(string(emailContentFile)))

type Config struct {
	From         string
	To           string
	Subject      string
	SMTPServer   string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
}

// Content : the data to include in the email
type Content struct {
	Devices interface{}
	Root    string
}

//SendUpdate : Send an email
func SendUpdate(config *Config, input *Content) error {
	var buffer bytes.Buffer
	err := emailContentTemplate.Execute(&buffer, *input)
	if err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", config.From)
	m.SetHeader("To", config.To)
	m.SetHeader("Subject", config.Subject)
	m.SetBody("text/html", buffer.String())

	d := gomail.NewDialer(config.SMTPServer, config.SMTPPort, config.SMTPUser, config.SMTPPassword)
	err = d.DialAndSend(m)
	return err
}
