package app

import (
	"net/url"
	"os"
	"strings"

	"github.com/metal-toolbox/hollow-bomservice/internal/model"
	"github.com/pkg/errors"
	"go.hollow.sh/toolbox/ginjwt"
)

var (
	ErrConfig = errors.New("configuration error")
)

// Configuration holds application configuration read from a YAML or set by env variables.
//
// nolint:govet // prefer readability over field alignment optimization for this case.
type Configuration struct {
	// file is the configuration file path
	file string `mapstructure:"-"`

	// LogLevel is the app verbose logging level.
	// one of - info, debug, trace
	LogLevel string `mapstructure:"log_level"`

	// ListenAddress is the server listen address
	ListenAddress string `mapstructure:"listen_address"`

	// APIServerJWTAuth sets the JWT verification configuration for the bomservice API service.
	APIServerJWTAuth *ginjwt.AuthConfig `mapstructure:"ginjwt_auth"`

	// APIOIDCOptions defines configuration to handle OIDC authn/authz for bomservice API clients.
	APIOIDCOptions APIOIDCOptions `mapstructure:"api_server_oidc"`

	// ServerserviceOptions defines the serverservice client configuration parameters
	ServerserviceOptions ServerserviceOptions `mapstructure:"serverservice"`
}

// APIOIDCOptions defines configuration to handle OIDC authn/authz for bomservice API clients.
type APIOIDCOptions struct {
	EnabledOAuth     bool   `mapstructure:"enable_oauth"`
	IssuerEndpoint   string `mapstructure:"issuer_endpoint"`
	AudienceEndpoint string `mapstructure:"audience_endpoint"`
	JWKSURI          string `mapstructure:"jwksuri"`
	RolesClaim       string `mapstructure:"roles_claim"`
	UsernameClaim    string `mapstructure:"username_claim"`
}

// ServerserviceOptions defines configuration for the Serverservice client.
// https://github.com/metal-toolbox/hollow-serverservice
type ServerserviceOptions struct {
	EndpointURL          *url.URL
	Endpoint             string   `mapstructure:"endpoint"`
	OidcIssuerEndpoint   string   `mapstructure:"oidc_issuer_endpoint"`
	OidcAudienceEndpoint string   `mapstructure:"oidc_audience_endpoint"`
	OidcClientSecret     string   `mapstructure:"oidc_client_secret"`
	OidcClientID         string   `mapstructure:"oidc_client_id"`
	OidcClientScopes     []string `mapstructure:"oidc_client_scopes"`
	DisableOAuth         bool     `mapstructure:"disable_oauth"`
}

func (a *App) LoadConfiguration() error {
	a.v.SetConfigType("yaml")
	a.v.SetEnvPrefix(model.AppName)
	a.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	a.v.AutomaticEnv()

	fh, err := os.Open(a.Config.file)
	if err != nil {
		return errors.Wrap(err, ErrConfig.Error())
	}

	if err = a.v.ReadConfig(fh); err != nil {
		return errors.Wrap(err, ErrConfig.Error()+" "+a.Config.file)
	}

	if err := a.v.Unmarshal(a.Config); err != nil {
		return errors.Wrap(err, a.Config.file)
	}

	if err := a.envVarOverrides(); err != nil {
		return errors.Wrap(err, ErrConfig.Error())
	}

	return nil
}

func (a *App) envVarOverrides() error {
	if a.v.GetString("log.level") != "" {
		a.Config.LogLevel = a.v.GetString("log.level")
	}

	if a.v.GetString("app.kind") != "" {
		a.AppKind = model.AppKind(a.v.GetString("app.kind"))
	}

	if a.v.GetString("listen.address") != "" {
		a.Config.ListenAddress = a.v.GetString("listen.address")
	}

	if a.v.GetString("store.kind") != "" {
		a.Config.ListenAddress = a.v.GetString("store.kind")
	}
	if err := a.envVarServerserviceOverrides(); err != nil {
		return err
	}

	if err := a.apiServerJWTAuthParams(); err != nil {
		return errors.Wrap(ErrConfig, err.Error())
	}

	return nil
}

func (a *App) apiServerJWTAuthParams() error {
	if !a.v.GetBool("api.oidc.enabled") {
		return nil
	}

	errOIDCAuthParams := errors.New("bomservice API OIDC Auth params not defined")

	required := []string{
		"audience.endpoint",
		"issuer.endpoint",
		"jwksuri",
		"claims.roles",
		"claims.username",
	}

	var unset []string

	for _, k := range required {
		if a.v.GetString("api.oidc."+k) == "" {
			unset = append(unset, "api.oidc."+k)
		}
	}

	if len(unset) > 0 {
		return errors.Wrap(errOIDCAuthParams, strings.Join(unset, ","))
	}

	a.Config.APIServerJWTAuth = &ginjwt.AuthConfig{
		Enabled:       true,
		Audience:      a.v.GetString("api.oidc.audience.endpoint"),
		Issuer:        a.v.GetString("api.oidc.issuer.endpoint"),
		JWKSURI:       a.v.GetString("api.oidc.jwksuri"),
		LogFields:     a.v.GetStringSlice("api.oidc.log"),
		RolesClaim:    a.v.GetString("api.oidc.claims.roles"),
		UsernameClaim: a.v.GetString("api.oidc.claims.username"),
	}

	return nil
}

// Server service configuration options

// nolint:gocyclo // parameter validation is cyclomatic
func (a *App) envVarServerserviceOverrides() error {
	if a.v.GetString("serverservice.endpoint") != "" {
		a.Config.ServerserviceOptions.Endpoint = a.v.GetString("serverservice.endpoint")
	}

	endpointURL, err := url.Parse(a.Config.ServerserviceOptions.Endpoint)
	if err != nil {
		return errors.New("serverservice endpoint URL error: " + err.Error())
	}

	a.Config.ServerserviceOptions.EndpointURL = endpointURL

	if a.v.GetString("serverservice.disable.oauth") != "" {
		a.Config.ServerserviceOptions.DisableOAuth = a.v.GetBool("serverservice.disable.oauth")
	}

	if a.Config.ServerserviceOptions.DisableOAuth {
		return nil
	}

	if a.v.GetString("serverservice.oidc.issuer.endpoint") != "" {
		a.Config.ServerserviceOptions.OidcIssuerEndpoint = a.v.GetString("serverservice.oidc.issuer.endpoint")
	}

	if a.Config.ServerserviceOptions.OidcIssuerEndpoint == "" {
		return errors.New("serverservice oidc.issuer.endpoint not defined")
	}

	if a.v.GetString("serverservice.oidc.audience.endpoint") != "" {
		a.Config.ServerserviceOptions.OidcAudienceEndpoint = a.v.GetString("serverservice.oidc.audience.endpoint")
	}

	if a.Config.ServerserviceOptions.OidcAudienceEndpoint == "" {
		return errors.New("serverservice oidc.audience.endpoint not defined")
	}

	if a.v.GetString("serverservice.oidc.client.secret") != "" {
		a.Config.ServerserviceOptions.OidcClientSecret = a.v.GetString("serverservice.oidc.client.secret")
	}

	if a.Config.ServerserviceOptions.OidcClientSecret == "" {
		return errors.New("serverservice oidc.client.secret not defined")
	}

	if a.v.GetString("serverservice.oidc.client.id") != "" {
		a.Config.ServerserviceOptions.OidcClientID = a.v.GetString("serverservice.oidc.client.id")
	}

	if a.Config.ServerserviceOptions.OidcClientID == "" {
		return errors.New("serverservice oidc.client.id not defined")
	}

	if a.v.GetString("serverservice.oidc.client.scopes") != "" {
		a.Config.ServerserviceOptions.OidcClientScopes = a.v.GetStringSlice("serverservice.oidc.client.scopes")
	}

	if len(a.Config.ServerserviceOptions.OidcClientScopes) == 0 {
		return errors.New("serverservice oidc.client.scopes not defined")
	}

	return nil
}
