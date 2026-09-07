package main

import (
	"fmt"
	"os"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/goccy/go-yaml"
)

type Config struct {
	App  *kingpin.Application
	Root *RootConfig
	// File is the path of the configuration file, kept for reloading.
	File string
	// CLI flag overrides, re-applied on every reload.
	mailURL      string
	mailUsername string
	mailPassword string
}

type RootConfig struct {
	Host      string        `yaml:"host,omitempty"`
	StaticDir string        `yaml:"static_dir,omitempty"`
	HTTP      HTTPConfig    `yaml:"http,omitempty"`
	Mail      MailConfig    `yaml:"mail,omitempty"`
	Routes    []RouteConfig `yaml:"routes,omitempty"`
}

type HTTPConfig struct {
	ListenOn        string        `yaml:"listen_on,omitempty"`
	UseProxyProto   bool          `yaml:"use_proxyproto,omitempty"`
	NotFound        string        `yaml:"not_found,omitempty"`
	CacheExpiration time.Duration `yaml:"cache_expiration,omitempty"`
	CacheControl    bool          `yaml:"cache_control,omitempty"`
}

type MailConfig struct {
	MailURL           string `yaml:"mail_url,omitempty"`
	MailTLSSkipVerify bool   `yaml:"mail_tls_skip_verify,omitempty"`
	MailUsername      string `yaml:"mail_username,omitempty"`
	MailPassword      string `yaml:"mail_password,omitempty"`
}

type RouteConfig struct {
	Path         string `yaml:"path,omitempty"`
	Method       string `yaml:"method,omitempty"`
	MailTemplate string `yaml:"mail_template,omitempty"`
	// TODO implement
	MailTemplateFile  string            `yaml:"mail_template_file,omitempty"`
	MailSubject       string            `yaml:"mail_subject,omitempty"`
	MailTo            []string          `yaml:"mail_to,omitempty"`
	MailFrom          string            `yaml:"mail_from,omitempty"`
	FormFieldsMapping map[string]string `yaml:"form_fields_mapping,omitempty"`
	Response          ResponseConfig    `yaml:"response,omitempty"`
}

type ResponseConfig struct {
	HTMLTag    string `yaml:"html_tag,omitempty"`
	Success    string `yaml:"success,omitempty"`
	Validation string `yaml:"validation,omitempty"`
	Failure    string `yaml:"failure,omitempty"`
}

// NewConfig creates a new app configuration.
func NewConfig() *Config {
	app := kingpin.New("ws", "Website Server with basic email client for contact forms").DefaultEnvars()

	configFile := app.Flag("config", "Path to the configuration file").Short('c').Default("config.yaml").String()
	mailURL := app.Flag("mail-url", "URL for the mail server").Default("").String()
	mailUsername := app.Flag("mail-username", "Username for the mail server").Default("").String()
	mailPassword := app.Flag("mail-password", "Password for the mail server").Default("").String()

	app.Version(version).VersionFlag.Short('v').NoEnvar()
	app.HelpFlag.Short('h').NoEnvar()
	kingpin.MustParse(app.Parse(os.Args[1:]))

	config := &Config{
		App:          app,
		File:         *configFile,
		mailURL:      *mailURL,
		mailUsername: *mailUsername,
		mailPassword: *mailPassword,
	}

	if len(config.File) > 0 {
		if err := config.LoadFromYAML(config.File); err != nil {
			app.Fatalf("failed to load config file: %s", err)
		}
	}
	config.applyOverrides()

	return config
}

// Reload re-reads the configuration file and re-applies the CLI flag overrides.
func (c *Config) Reload() error {
	if len(c.File) == 0 {
		return fmt.Errorf("no config file to reload")
	}
	if err := c.LoadFromYAML(c.File); err != nil {
		return err
	}
	c.applyOverrides()
	return nil
}

// applyOverrides applies CLI flag overrides on top of the loaded configuration.
func (c *Config) applyOverrides() {
	if len(c.mailUsername) > 0 {
		c.Root.Mail.MailUsername = c.mailUsername
	}
	if len(c.mailPassword) > 0 {
		c.Root.Mail.MailPassword = c.mailPassword
	}
	if len(c.mailURL) > 0 {
		c.Root.Mail.MailURL = c.mailURL
	}
}

// LoadFromYAML loads the root configuration from a YAML file.
func (c *Config) LoadFromYAML(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	d := yaml.NewDecoder(f)
	rc := &RootConfig{}
	err = d.Decode(rc)
	if err != nil {
		return err
	}
	c.Root = rc

	return nil
}
