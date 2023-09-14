package client

import (
	"context"
	"fmt"
	"net/http"

	sservice "go.hollow.sh/serverservice/pkg/api/v1"
)

const (
	bomInfoEndpoint            = "bomservice"
	uploadFileEndpoint         = "upload-xlsx-file"
	bomByMacAOCAddressEndpoint = "aoc-mac-address"
	bomByMacBMCAddressEndpoint = "bmc-mac-address"
)

// Doer performs HTTP requests.
//
// The standard http.Client implements this interface.
type HTTPRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client can perform queries against the Bom service.
type Client struct {
	// The server address with the schema
	serverAddress string

	// Authentication token
	authToken string

	// Doer for performing requests, typically a *http.Client with any
	// customized settings, such as certificate chains.
	client HTTPRequestDoer
}

// Option allows setting custom parameters during construction
type Option func(*Client) error

// Creates a new Client, with reasonable defaults
func NewClient(serverAddress string, opts ...Option) (*Client, error) {
	// create a client with sane default values
	client := Client{serverAddress: serverAddress}
	// mutate client and add all optional params
	for _, o := range opts {
		if err := o(&client); err != nil {
			return nil, err
		}
	}

	// create httpClient, if not already present
	if client.client == nil {
		client.client = &http.Client{}
	}

	return &client, nil
}

// WithHTTPClient allows overriding the default Doer, which is
// automatically created using http.Client. This is useful for tests.
func WithHTTPClient(doer HTTPRequestDoer) Option {
	return func(c *Client) error {
		c.client = doer
		return nil
	}
}

// WithAuthToken sets the client auth token.
func WithAuthToken(authToken string) Option {
	return func(c *Client) error {
		c.authToken = authToken
		return nil
	}
}

func (c *Client) XlsxFileUpload(ctx context.Context, fileBytes []byte) (*sservice.ServerResponse, error) {
	path := fmt.Sprintf("%s/%s", bomInfoEndpoint, uploadFileEndpoint)
	return c.postRawBytes(ctx, path, fileBytes)
}

func (c *Client) GetBomInfoByAOCMacAddr(ctx context.Context, aocMacAddr string) (*sservice.ServerResponse, error) {
	path := fmt.Sprintf("%s/%s/%s", bomInfoEndpoint, bomByMacAOCAddressEndpoint, aocMacAddr)
	return c.get(ctx, path)
}

func (c *Client) GetBomInfoByBMCMacAddr(ctx context.Context, bmcMacAddr string) (*sservice.ServerResponse, error) {
	path := fmt.Sprintf("servers/%s/condition/%s", bomByMacBMCAddressEndpoint, bmcMacAddr)
	return c.get(ctx, path)
}
