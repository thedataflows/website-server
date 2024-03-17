package main

import (
	"crypto/tls"
	"html/template"
	"net/url"
	"strconv"

	"github.com/wneessen/go-mail"
)

type mailSenderConf struct {
	mailServer            string
	mailPort              int
	userAgent             string
	tlsInsecureSkipVerify bool
	username              string
	password              string
}

// newMailSender creates a new mail configuration.
func newMailSender(mailUrl string) (*mailSenderConf, error) {
	ms := &mailSenderConf{}

	if len(mailUrl) == 0 {
		mailUrl = "smtp://localhost:1025"
	}

	url, err := url.Parse(mailUrl)
	if err != nil {
		return nil, err
	}

	ms.mailServer = url.Hostname()
	if len(ms.mailServer) == 0 {
		ms.mailServer = "localhost"
	}

	ms.mailPort, err = strconv.Atoi(url.Port())
	if err != nil {
		ms.mailPort = 1025
	}

	if len(ms.username) == 0 {
		ms.username = url.User.Username()
	}
	if len(ms.password) == 0 {
		ms.password, _ = url.User.Password()
	}

	return ms, nil
}

// NewMessageFromTemplate creates a new mail message from a template.
func (ms *mailSenderConf) NewMessageFromTemplate(subject, from string, to []string, tpl string, data map[string]string) (*mail.Msg, error) {
	m := mail.NewMsg()
	m.SetGenHeader(mail.HeaderUserAgent, ms.userAgent)
	m.SetGenHeader(mail.HeaderXMailer, ms.userAgent)

	m.Subject(subject)
	if err := m.From(from); err != nil {
		return nil, err
	}
	if err := m.To(to...); err != nil {
		return nil, err
	}

	tmpl, err := template.New("contact").Parse(tpl)
	if err != nil {
		return nil, err
	}

	err = m.SetBodyHTMLTemplate(tmpl, data)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// SendMail sends a mail message.
func (ms *mailSenderConf) SendMail(msg ...*mail.Msg) error {
	if len(msg) == 0 {
		return nil
	}

	c, err := mail.NewClient(
		ms.mailServer,
		mail.WithTLSPortPolicy(mail.TLSOpportunistic),
		mail.WithTLSConfig(
			&tls.Config{
				InsecureSkipVerify: ms.tlsInsecureSkipVerify,
			},
		),
		mail.WithPort(ms.mailPort),
	)
	if err != nil {
		return err
	}
	defer c.Close()

	if len(ms.username) > 0 {
		c.SetUsername(ms.username)
	}
	if len(ms.password) > 0 {
		c.SetPassword(ms.password)
	}

	for _, m := range msg {
		if err := c.DialAndSend(m); err != nil {
			return err
		}
	}

	return nil
}
