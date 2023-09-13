package routes

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/metal-toolbox/hollow-bomservice/internal/store"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	sservice "go.hollow.sh/serverservice/pkg/api/v1"
	"go.hollow.sh/toolbox/ginjwt"
)

const (
	PathPrefix = "/api/v1"
)

var ginNoOp = func(_ *gin.Context) {}

// Bom provides a struct to map the bom_info table.
// Naming conversion is strange here just in order to make it consistent
// with generated BomInfo.
type Bom struct {
	SerialNum     string `json:"serial_num"`
	AocMacAddress string `json:"aoc_mac_address"`
	BmcMacAddress string `json:"bmc_mac_address"`
	NumDefiPmi    string `json:"num_defi_pmi"`
	NumDefPWD     string `json:"num_def_pwd"`
	Metro         string `json:"metro"`
}

// AocMacAddressBom provides a struct to map the aoc_mac_address table.
type AocMacAddressBom struct {
	AocMacAddress string `json:"aoc_mac_address"`
	SerialNum     string `json:"serial_num"`
}

// Routes type sets up the bomservice API  router routes.
type Routes struct {
	authMW     *ginjwt.Middleware
	repository store.Repository
	logger     *logrus.Logger
}

// Option type sets a parameter on the Routes type.
type Option func(*Routes)

// WithStore sets the storage repository on the routes type.
func WithStore(repository store.Repository) Option {
	return func(r *Routes) {
		r.repository = repository
	}
}

// WithLogger sets the logger on the routes type.
func WithLogger(logger *logrus.Logger) Option {
	return func(r *Routes) {
		r.logger = logger
	}
}

// WithAuthMiddleware sets the auth middleware on the routes type.
func WithAuthMiddleware(authMW *ginjwt.Middleware) Option {
	return func(r *Routes) {
		r.authMW = authMW
	}
}

// apiHandler is a function that performs real work for the bomservice API
type apiHandler func(c *gin.Context) (int, *sservice.ServerResponse)

// wrapAPICall wraps a bomservice routine that does work with some prometheus
// metrics collection and returns a gin.HandlerFunc so the middleware can execute
// directly
func wrapAPICall(fn apiHandler) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		responseCode, obj := fn(ctx)
		ctx.JSON(responseCode, obj)
	}
}

// NewRoutes returns a new bomservice API routes with handlers registered.
func NewRoutes(options ...Option) (*Routes, error) {
	routes := &Routes{}

	for _, opt := range options {
		opt(routes)
	}

	supported := []string{}

	if routes.repository == nil {
		return nil, errors.Wrap(ErrStore, "no store repository defined")
	}

	routes.logger.Debug(
		"routes initialized with support for bomservice: ",
		strings.Join(supported, ","),
	)

	return routes, nil
}

func (r *Routes) composeAuthHandler(scopes []string) gin.HandlerFunc {
	if r.authMW == nil {
		return ginNoOp
	}
	return r.authMW.RequiredScopes(scopes)
}

func (r *Routes) Routes(g *gin.RouterGroup) {
	// JWT token verification.
	if r.authMW != nil {
		g.Use(r.authMW.AuthRequired())
	}

	bomService := g.Group("/bomservice")
	{
		bomService.POST("/upload-xlsx-file",
			r.composeAuthHandler(createScopes("upload-xlsx-file")),
			wrapAPICall(r.billOfMaterialsBatchUpload))

		bomService.GET("/aoc-mac-address/:aoc_mac_address",
			r.composeAuthHandler(readScopes("aoc-mac-address")),
			wrapAPICall(r.getBomInfoByAOCMacAddr))

		bomService.GET("/bmc-mac-address/:bmc_mac_address",
			r.composeAuthHandler(readScopes("bmc-mac-address")),
			wrapAPICall(r.getBomInfoByBMCMacAddr))
	}
}

func createScopes(items ...string) []string {
	s := []string{"write", "create"}
	for _, i := range items {
		s = append(s, fmt.Sprintf("create:%s", i))
	}

	return s
}

func readScopes(items ...string) []string {
	s := []string{"read"}
	for _, i := range items {
		s = append(s, fmt.Sprintf("read:%s", i))
	}

	return s
}
