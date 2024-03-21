package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/pires/go-proxyproto"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	EMAIL_FIELD   = "email"
	MAX_FIELD_LEN = 254
	EMAIL_REGEX   = `^[a-zA-Z0-9]+[._\%+\-]*[^.]*@[a-z0-9.\-]+\.[a-z]{2,}$`
)

var (
	version           = "dev"
	ErrNonHtmxRequest = fiber.NewError(fiber.StatusBadRequest, "non-htmx request")
	ErrInvalidInput   = fiber.NewError(fiber.StatusBadRequest, "Invalid input")
)

// FrontendServer is the main server object.
type FrontendServer struct {
	config *Config
	logger zerolog.Logger
}

type routeSpec struct {
	parent      *FrontendServer
	routeConfig RouteConfig
	mailSender  mailSenderConf
}

// loggingMiddleware logs the request.
func (srv *FrontendServer) loggingMiddleware(c *fiber.Ctx) error {
	// Start timer
	start := time.Now()
	// Next middleware
	err := c.Next()
	// Stop timer
	stop := time.Now()
	// Log the request
	if err != nil {
		errCode := 500
		if e, ok := err.(*fiber.Error); ok {
			errCode = e.Code
		}
		srv.logger.
			Err(err).
			Str("ip", c.IP()).
			Str("met", c.Method()).
			Int("sta", errCode).
			Int64("dur", stop.Sub(start).Milliseconds()).
			Msg(c.Path())
		return err
	}
	srv.logger.
		Info().
		Str("ip", c.IP()).
		Str("met", c.Method()).
		Int("sta", c.Response().StatusCode()).
		Int64("dur", stop.Sub(start).Milliseconds()).
		Msg(c.Path())

	// Return the error, if any
	return nil
}

// errorHandlingMiddleware handles server errors
func (srv *FrontendServer) errorHandlingMiddleware(c *fiber.Ctx) error {
	err := c.Next()

	if err != nil {
		if e, ok := err.(*fiber.Error); ok {
			switch e.Code {
			case fiber.StatusNotFound:
				return c.Redirect(srv.config.Root.HTTP.NotFound, e.Code)
			}
		}
		c.Status(http.StatusInternalServerError)
	}

	return err
}

// isHtmxRequest checks if the request is an htmx request.
func isHtmxRequest(c *fiber.Ctx) error {
	if c.Get("HX-Request") == "" || c.Get("HX-Request") != "true" {
		return ErrNonHtmxRequest
	}
	return nil
}

// fiberResponse is a generic response for a fiber request.
func fiberResponse(c *fiber.Ctx, status int, message, format string) error {
	c.Status(status)
	if format == "" {
		format = "<div id=\"htmxresponse\">%s</div>"
	}
	c.Response().SetBodyString(fmt.Sprintf(format, message))
	return nil
}

func (rs *routeSpec) buildTemplateMap(c *fiber.Ctx, fieldsMap map[string]string, format string) map[string]string {
	m := make(map[string]string)
	m["_SiteURL"] = c.Hostname()
	m["_IP"] = c.IP()

	re, _ := regexp.Compile(EMAIL_REGEX)
	for k, v := range fieldsMap {
		m[k] = strings.TrimSpace(c.FormValue(v))
		if len(m[k]) == 0 || len(m[k]) > MAX_FIELD_LEN {
			_ = fiberResponse(c, ErrInvalidInput.Code, rs.routeConfig.Response.Validation, format)
			return nil
		}
		// special case for email
		if v == EMAIL_FIELD {
			// Remove the +... part from the email to prevent multiple registrations with the same email
			m[k] = regexp.MustCompile(`\+[^@]*`).ReplaceAllString(m[k], "")
			// Validate email using regex
			if !re.MatchString(m[k]) {
				// Do not give specific feedback on purpose to avoid leaking information.
				_ = fiberResponse(c, http.StatusInternalServerError, rs.routeConfig.Response.Validation, format)
				return nil
			}
		}
	}

	return m
}

// postHandler handles a POST request assumed to be contact.
func (rs *routeSpec) postHandler(c *fiber.Ctx) error {
	if err := isHtmxRequest(c); err != nil {
		return err
	}

	templateMap := rs.buildTemplateMap(c, rs.routeConfig.FormFieldsMapping, rs.routeConfig.Response.HTMLTag)
	if templateMap == nil {
		return nil
	}
	// Send the contact email
	msg, err := rs.mailSender.NewMessageFromTemplate(
		fmt.Sprintf("%s - %s", rs.routeConfig.MailSubject, rs.parent.config.Root.Host),
		rs.routeConfig.MailFrom,
		rs.routeConfig.MailTo,
		rs.routeConfig.MailTemplate,
		templateMap,
	)
	if err != nil {
		rs.parent.logger.
			Err(err).
			Msg(rs.routeConfig.Response.Failure)
		return err
	}
	err = rs.mailSender.SendMail(msg)
	if err != nil {
		rs.parent.logger.
			Err(err).
			Msg(rs.routeConfig.Response.Failure)
		return fiberResponse(c, http.StatusInternalServerError, rs.routeConfig.Response.Failure, rs.routeConfig.Response.HTMLTag)
	}

	return fiberResponse(c, http.StatusOK, rs.routeConfig.Response.Success, rs.routeConfig.Response.HTMLTag)
}

// NewFiber creates a new Fiber app with optional Fiber configurations: if specified, only the first one is used.
func (srv *FrontendServer) NewFiber() *fiber.App {
	fiberConfig := fiber.Config{
		// For more information, see https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
		ReadTimeout:           1 * time.Second,
		WriteTimeout:          3 * time.Second,
		DisableStartupMessage: true,
	}

	// Create a new Fiber server.
	app := fiber.New(fiberConfig)

	// Add Fiber middlewares. Order is relevant!
	app.Use(srv.errorHandlingMiddleware)
	app.Use(srv.loggingMiddleware)
	app.Use(cache.New(cache.Config{
		Next: func(c *fiber.Ctx) bool {
			return c.Query("noCache") == "true"
		},
		Expiration:   srv.config.Root.HTTP.CacheExpiration,
		CacheControl: srv.config.Root.HTTP.CacheControl,
	}))
	app.Use(etag.New(etag.Config{
		Weak: true,
	}))
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed, // 1
	}))

	// Handle static files.
	app.Static("/", srv.config.Root.StaticDir)
	srv.logger.Info().Msgf("Serving static files from '%s'", srv.config.Root.StaticDir)

	ms, err := newMailSender(srv.config.Root.Mail.MailURL)
	if err != nil {
		srv.logger.Fatal().Err(err).Msg("failed to create mail sender")
	}
	ms.userAgent = srv.config.Root.Host
	ms.tlsInsecureSkipVerify = srv.config.Root.Mail.MailTLSSkipVerify
	ms.username = srv.config.Root.Mail.MailUsername
	ms.password = srv.config.Root.Mail.MailPassword

	for _, route := range srv.config.Root.Routes {
		rs := &routeSpec{
			parent:      srv,
			routeConfig: route,
			mailSender:  *ms,
		}
		switch route.Method {
		// case "GET":
		// 	app.Get(route.Path, func(c *fiber.Ctx) error {
		// 		return c.SendFile(route.File)
		// 	})
		case "POST":
			app.Post(route.Path, rs.postHandler)
		}
	}

	return app
}

// Listen runs the server with optional Fiber configurations: if specified, only the first one is used.
func (srv *FrontendServer) Listen() (*fiber.App, error) {
	app := srv.NewFiber()

	if srv.config.Root.HTTP.UseProxyProto {
		listener, err := net.Listen("tcp", srv.config.Root.HTTP.ListenOn)
		if err != nil {
			return nil, err
		}
		listener = &proxyproto.Listener{
			Listener:          listener,
			ReadHeaderTimeout: 1 * time.Second,
		}
		err = app.Listener(listener)
		return app, err
	}

	err := app.Listen(srv.config.Root.HTTP.ListenOn)
	return app, err
}

// main is the entry point of the application.
func main() {
	config := NewConfig()

	srv := &FrontendServer{
		config: config,
		logger: log.Output(
			zerolog.ConsoleWriter{
				Out:        os.Stdout,
				TimeFormat: time.RFC3339Nano,
				NoColor:    false,
			},
		),
	}

	srv.logger.Info().Msgf("Starting website server '%s' '%s' on '%s'", config.Root.Host, version, config.Root.HTTP.ListenOn)
	_, err := srv.Listen()
	if err != nil {
		srv.logger.Fatal().Err(err).Msg("failed to start server")
	}
}
