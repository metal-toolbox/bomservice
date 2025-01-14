package store

import (
	"context"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/metal-toolbox/bomservice/internal/app"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	fleetdbapi "github.com/metal-toolbox/fleetdb/pkg/api/v1"
)

// Serverservice implements the Repository interface to have server bom objects stored as server attributes.
type Serverservice struct {
	config *app.ServerserviceOptions
	client *fleetdbapi.Client
	logger *logrus.Logger
}

var (
	// ErrServerserviceConfig is returned when theres an error in loading serverservice configuration.
	ErrServerserviceConfig = errors.New("Serverservice configuration error")

	// ErrServserviceAttribute is returned when a serverservice attribute does not contain the expected fields.
	ErrServerserviceAttribute = errors.New("error in serverservice attribute")

	// connectionTimeout is the maximum amount of time spent on each http connection to serverservice.
	connectionTimeout = 30 * time.Second
)

func newServerserviceStore(ctx context.Context, config *app.ServerserviceOptions, logger *logrus.Logger) (Repository, error) {
	s := &Serverservice{logger: logger, config: config}

	var client *fleetdbapi.Client
	var err error

	if !config.DisableOAuth {
		client, err = newClientWithOAuth(ctx, config, logger)
		if err != nil {
			return nil, err
		}
	} else {
		client, err = fleetdbapi.NewClientWithToken("fake", config.Endpoint, nil)
		if err != nil {
			return nil, err
		}
	}

	s.client = client

	return s, nil
}

// returns a serverservice retryable http client with Otel and Oauth wrapped in
func newClientWithOAuth(ctx context.Context, cfg *app.ServerserviceOptions, logger *logrus.Logger) (*fleetdbapi.Client, error) {
	// init retryable http client
	retryableClient := retryablehttp.NewClient()

	// disable default debug logging on the retryable client
	if logger.Level < logrus.DebugLevel {
		retryableClient.Logger = nil
	} else {
		retryableClient.Logger = logger
	}

	// setup oidc provider
	provider, err := oidc.NewProvider(ctx, cfg.OidcIssuerEndpoint)
	if err != nil {
		return nil, err
	}

	clientID := "bomservice-api"

	if cfg.OidcClientID != "" {
		clientID = cfg.OidcClientID
	}

	// setup oauth configuration
	oauthConfig := clientcredentials.Config{
		ClientID:       clientID,
		ClientSecret:   cfg.OidcClientSecret,
		TokenURL:       provider.Endpoint().TokenURL,
		Scopes:         cfg.OidcClientScopes,
		EndpointParams: url.Values{"audience": []string{cfg.OidcAudienceEndpoint}},
		// with this the oauth client spends less time identifying the client grant mechanism.
		AuthStyle: oauth2.AuthStyleInParams,
	}

	// wrap OAuth transport, cookie jar in the retryable client
	oAuthclient := oauthConfig.Client(ctx)

	retryableClient.HTTPClient.Transport = oAuthclient.Transport
	retryableClient.HTTPClient.Jar = oAuthclient.Jar

	httpClient := retryableClient.StandardClient()
	httpClient.Timeout = connectionTimeout

	return fleetdbapi.NewClientWithToken(
		cfg.OidcClientSecret,
		cfg.Endpoint,
		httpClient,
	)
}

// BillOfMaterialsBatchUpload will attempt to write multiple boms to database.
func (s *Serverservice) BillOfMaterialsBatchUpload(ctx context.Context, boms []fleetdbapi.Bom) (*fleetdbapi.ServerResponse, error) {
	return s.client.BillOfMaterialsBatchUpload(ctx, boms)
}

// GetBomInfoByAOCMacAddr will return the bom info object by the aoc mac address.
func (s *Serverservice) GetBomInfoByAOCMacAddr(ctx context.Context, macAddr string) (*fleetdbapi.Bom, *fleetdbapi.ServerResponse, error) {
	return s.client.GetBomInfoByAOCMacAddr(ctx, macAddr)
}

// GetBomInfoByBMCMacAddr will return the bom info object by the bmc mac address.
func (s *Serverservice) GetBomInfoByBMCMacAddr(ctx context.Context, macAddr string) (*fleetdbapi.Bom, *fleetdbapi.ServerResponse, error) {
	return s.client.GetBomInfoByBMCMacAddr(ctx, macAddr)
}
