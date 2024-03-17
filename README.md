# Website Server

Basic and opinionated website server with email sending capabilities for endpoints like contact forms.

## Usage

```ini
usage: ws [<flags>]

Website Server with basic email client for contact forms

Flags:
  -h, --[no-]help             Show context-sensitive help (also try --help-long
                              and --help-man).
      --config="config.yaml"  Path to the configuration file ($WS_CONFIG)
      --mail-url=""           URL for the mail server ($WS_MAIL_URL)
      --mail-username=""      Username for the mail server ($WS_MAIL_USERNAME)
      --mail-password=""      Password for the mail server ($WS_MAIL_PASSWORD)
  -v, --[no-]version          Show application version.
```

## Development

- go 1.22
- [mailpit](https://github.com/axllent/mailpit) (for testing emails)
- docker/podman or compatible
- [task](https://taskfile.dev)
  - List available tasks to run: `task`

## TODO

- [ ] Add tests
- [ ] Use a template file for the email body, like [contact.html](./contact.html)

## License

MIT
