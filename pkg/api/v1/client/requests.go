package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/metal-toolbox/hollow-bomservice/pkg/api/v1/routes"
	sservice "go.hollow.sh/serverservice/pkg/api/v1"
)

func (c *Client) get(ctx context.Context, path string) (*sservice.ServerResponse, error) {
	requestURL, err := url.Parse(fmt.Sprintf("%s%s/%s", c.serverAddress, routes.PathPrefix, path))
	if err != nil {
		return nil, Error{Cause: err.Error()}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), http.NoBody)
	if err != nil {
		return nil, Error{Cause: "error in GET request" + err.Error()}
	}

	return c.do(req)
}

func (c *Client) postRawBytes(ctx context.Context, path string, body []byte) (*sservice.ServerResponse, error) {
	requestURL, err := url.Parse(fmt.Sprintf("%s%s/%s", c.serverAddress, routes.PathPrefix, path))
	if err != nil {
		return nil, Error{Cause: err.Error()}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL.String(), bytes.NewReader(body))
	if err != nil {
		return nil, Error{Cause: "error in POST request" + err.Error()}
	}

	return c.do(req)
}

func (c *Client) do(req *http.Request) (*sservice.ServerResponse, error) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("bom-service-client"))

	if c.authToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("bearer %s", c.authToken))
	}

	response, err := c.client.Do(req)
	if err != nil {
		return nil, RequestError{err.Error(), c.statusCode(response)}
	}

	if response == nil {
		return nil, RequestError{"got empty response body", 0}
	}

	if response.StatusCode >= http.StatusMultiStatus {
		return nil, RequestError{"got bad request", c.statusCode(response)}
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, RequestError{
			"failed to read response body: " + err.Error(),
			c.statusCode(response),
		}
	}

	serverResponse := &sservice.ServerResponse{}

	if err := json.Unmarshal(data, &serverResponse); err != nil {
		return nil, RequestError{
			"failed to unmarshal response from server: " + err.Error(),
			c.statusCode(response),
		}
	}

	return serverResponse, nil
}

func (c *Client) statusCode(response *http.Response) int {
	if response != nil {
		return response.StatusCode
	}

	return 0
}
