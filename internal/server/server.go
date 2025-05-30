package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/metal-toolbox/bomservice/internal/store"
	"github.com/metal-toolbox/bomservice/pkg/api/v1/routes"
	"github.com/metal-toolbox/rivets/v2/ginjwt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	// Request read timeout.
	readTimeout = 10 * time.Second
	// Request write timeout.
	writeTimeout = 60 * time.Second

	ErrRoutes = errors.New("error in routes")
)

// Server type holds attributes of the bomservice server
type Server struct {
	// Logger is the app logger
	authMWConfig  *ginjwt.AuthConfig
	logger        *logrus.Logger
	listenAddress string
	repository    store.Repository
}

// Option type sets a parameter on the Server type.
type Option func(*Server)

// WithStore sets the storage repository on the Server type.
func WithStore(repository store.Repository) Option {
	return func(s *Server) {
		s.repository = repository
	}
}

// WithLogger sets the logger on the Server type.
func WithLogger(logger *logrus.Logger) Option {
	return func(s *Server) {
		s.logger = logger
	}
}

// WithListenAddress sets the Server listen address.
func WithListenAddress(addr string) Option {
	return func(s *Server) {
		s.listenAddress = addr
	}
}

// WithAuthMiddlewareConfig sets the auth middleware configuration.
func WithAuthMiddlewareConfig(authMWConfig *ginjwt.AuthConfig) Option {
	return func(s *Server) {
		s.authMWConfig = authMWConfig
	}
}

func New(opts ...Option) *http.Server {
	s := &Server{}

	for _, opt := range opts {
		opt(s)
	}

	g := gin.New()
	g.Use(loggerMiddleware(s.logger), gin.Recovery())

	options := []routes.Option{
		routes.WithLogger(s.logger),
		routes.WithStore(s.repository),
	}

	// add auth middleware
	if s.authMWConfig != nil && s.authMWConfig.Enabled {
		authMW, err := ginjwt.NewAuthMiddleware(*s.authMWConfig)
		if err != nil {
			s.logger.Fatal("failed to initialize auth middleware: ", "error", err)
		}

		options = append(options, routes.WithAuthMiddleware(authMW))
	}

	v1Router, err := routes.NewRoutes(options...)
	if err != nil {
		s.logger.Fatal(errors.Wrap(err, ErrRoutes.Error()))
	}

	v1Router.Routes(g.Group(routes.PathPrefix))

	g.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"message": "invalid request - route not found"})
	})

	return &http.Server{
		Addr:         s.listenAddress,
		Handler:      g,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}
}
