//go:build testtools
// +build testtools

// Note:
// The testtools build flag is defined on this file since its required for ginjwt helper methods.
// Make sure to include `-tags testtools` in the build flags to ensure the tests in this file are run.

// for example:
// /usr/local/bin/go test -timeout 10s -run ^TestIntegration_ConditionsGet$ \
//   -tags testtools github.com/metal-toolbox/conditionorc/pkg/api/v1/client -v

package client

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/metal-toolbox/bomservice/internal/server"
	mockstore "github.com/metal-toolbox/bomservice/internal/store/mock"
	fleetdbapi "github.com/metal-toolbox/fleetdb/pkg/api/v1"
	"github.com/metal-toolbox/rivets/ginjwt"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

type integrationTester struct {
	t               *testing.T
	assertAuthToken bool
	handler         http.Handler
	client          *Client
	repository      *mockstore.MockRepository
}

// Do implements the HTTPRequestDoer interface to swap the response writer
func (i *integrationTester) Do(req *http.Request) (*http.Response, error) {
	if err := req.Context().Err(); err != nil {
		return nil, err
	}

	w := httptest.NewRecorder()
	i.handler.ServeHTTP(w, req)

	if i.assertAuthToken {
		assert.NotEmpty(i.t, req.Header.Get("Authorization"))
	} else {
		assert.Empty(i.t, req.Header.Get("Authorization"))
	}

	return w.Result(), nil
}

func newTester(t *testing.T, enableAuth bool, authToken string) *integrationTester {
	t.Helper()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repository := mockstore.NewMockRepository(ctrl)

	l := logrus.New()
	l.Level = logrus.Level(logrus.ErrorLevel)
	serverOptions := []server.Option{
		server.WithLogger(l),
		server.WithListenAddress("localhost:9999"),
		server.WithStore(repository),
	}

	// setup JWT auth middleware on router when a non-empty auth token was provided
	if enableAuth {
		jwksURI := ginjwt.TestHelperJWKSProvider(ginjwt.TestPrivRSAKey1ID, ginjwt.TestPrivRSAKey2ID)

		serverOptions = append(serverOptions,
			server.WithAuthMiddlewareConfig(&ginjwt.AuthConfig{
				Enabled:  true,
				Issuer:   "conditionorc.oidc.issuer",
				Audience: "conditionorc.client",
				JWKSURI:  jwksURI,
			},
			),
		)

	}

	gin.SetMode(gin.ReleaseMode)

	srv := server.New(serverOptions...)

	// setup test server httptest recorder
	tester := &integrationTester{
		t:          t,
		handler:    srv.Handler,
		repository: repository,
	}

	// setup test client
	clientOptions := []Option{WithHTTPClient(tester)}

	if enableAuth {
		// enable auth token assert on the server
		tester.assertAuthToken = true

		// client to include Authorization header
		clientOptions = append(clientOptions, WithAuthToken(authToken))
	}

	client, err := NewClient("http://localhost:9999", clientOptions...)
	if err != nil {
		t.Error(err)
	}

	tester.client = client

	return tester
}

func testAuthToken(t *testing.T) string {
	t.Helper()

	claims := jwt.Claims{
		Subject:   "test-user",
		Issuer:    "bomservice.oidc.issuer",
		NotBefore: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		Expiry:    jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Audience:  jwt.Audience{"bomservice.client"},
	}
	signer := ginjwt.TestHelperMustMakeSigner(jose.RS256, ginjwt.TestPrivRSAKey1ID, ginjwt.TestPrivRSAKey1)

	token, err := jwt.Signed(signer).Claims(claims).CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}

	return token
}

func TestIntegrationUploadXlsxFile(t *testing.T) {
	tester := newTester(t, false, "")

	testcases := []struct {
		name                string
		mockStore           func(r *mockstore.MockRepository)
		filePath            string
		expectResponse      *fleetdbapi.ServerResponse
		expectError         bool
		expectErrorContains string
	}{
		{
			"valid response",
			func(r *mockstore.MockRepository) {
				var expectedboms []fleetdbapi.Bom = []fleetdbapi.Bom{
					{
						SerialNum:     "test-serial-1",
						AocMacAddress: "FakeAOC1,FakeAOC2",
						BmcMacAddress: "FakeMac1,FakeMac2",
						NumDefiPmi:    "FakeDEFI1",
						NumDefPWD:     "FakeDEFPWD1",
					},
				}
				r.EXPECT().
					BillOfMaterialsBatchUpload(
						gomock.Any(),
						gomock.Eq(expectedboms),
					).
					Return(
						&fleetdbapi.ServerResponse{},
						nil).
					Times(1)
			},
			"./../../../../internal/parse/testdata/test_valid_one_bom.xlsx",
			&fleetdbapi.ServerResponse{},
			false,
			"",
		},
		{
			"Bad Request 400 response since serial number col is empty",
			func(r *mockstore.MockRepository) {},
			"./../../../../internal/parse/testdata/test_empty_serial_col.xlsx",
			&fleetdbapi.ServerResponse{},
			true,
			"got bad request",
		},
		{
			"Bad Request 400 response since empty serial number",
			func(r *mockstore.MockRepository) {},
			"./../../../../internal/parse/testdata/test_empty_serial.xlsx",
			&fleetdbapi.ServerResponse{},
			true,
			"got bad request",
		},
		{
			"Bad Request 400 response since empty bmc mac address",
			func(r *mockstore.MockRepository) {},
			"./../../../../internal/parse/testdata/test_empty_bmcMacAddress.xlsx",
			&fleetdbapi.ServerResponse{},
			true,
			"got bad request",
		},
		{
			"Bad Request 400 response since empty aoc mac address",
			func(r *mockstore.MockRepository) {},
			"./../../../../internal/parse/testdata/test_empty_aocMacAddress.xlsx",
			&fleetdbapi.ServerResponse{},
			true,
			"got bad request",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.mockStore != nil {
				tc.mockStore(tester.repository)
			}

			file, err := os.Open(tc.filePath)
			if err != nil {
				t.Errorf("os.Open(%v) failed to open file %v", tc.filePath, err)
				return
			}

			// Translate file to bytes since ParseXlsxFile accepts bytes of file as argument,
			// which is the format of a file reading from the HTTP request.
			stat, err := file.Stat()
			if err != nil {
				t.Errorf("Failed to read file %v", err)
				return
			}

			bs := make([]byte, stat.Size())
			_, err = bufio.NewReader(file).Read(bs)
			if err != nil && err != io.EOF {
				t.Errorf("Failed to read file %v", err)
				return
			}

			got, err := tester.client.XlsxFileUpload(context.TODO(), bs)
			if tc.expectError {
				if err == nil {
					t.Errorf("XlsxFileUpload(%v) expected error %v, got nil", tc.filePath, tc.expectErrorContains)
					return
				}
				if !strings.Contains(err.Error(), tc.expectErrorContains) {
					t.Errorf("XlsxFileUpload(%v) expect error message %v, got %v", tc.filePath, tc.expectErrorContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("XlsxFileUpload(%v) got err %v, expect nil", tc.filePath, err)
				return
			}

			if !reflect.DeepEqual(got, tc.expectResponse) {
				t.Errorf("XlsxFileUpload(%v) receives %+v, expects %+v", tc.filePath, got, tc.expectResponse)
			}
		})
	}
}

func TestIntegrationGetByAOCMacAddr(t *testing.T) {
	tester := newTester(t, false, "")

	testcases := []struct {
		name                string
		mockStore           func(r *mockstore.MockRepository)
		aocMacAddr          string
		expectResponse      *fleetdbapi.ServerResponse
		expectError         bool
		expectErrorContains string
	}{
		{
			"valid response",
			func(r *mockstore.MockRepository) {
				fakeFoundBom := fleetdbapi.Bom{
					SerialNum:     "test-serial-1",
					AocMacAddress: "FakeAOC1,FakeAOC2",
					BmcMacAddress: "FakeMac1,FakeMac2",
					NumDefiPmi:    "FakeDEFI1",
					NumDefPWD:     "FakeDEFPWD1",
				}
				r.EXPECT().
					GetBomInfoByAOCMacAddr(
						gomock.Any(),
						gomock.Eq("FakeAOC1"),
					).
					Return(
						&fakeFoundBom,
						&fleetdbapi.ServerResponse{
							Record: fakeFoundBom,
						},
						nil).
					Times(1)
			},
			"FakeAOC1",
			&fleetdbapi.ServerResponse{Record: interface{}(
				map[string]interface{}{
					"aoc_mac_address": "FakeAOC1,FakeAOC2",
					"bmc_mac_address": "FakeMac1,FakeMac2",
					"metro":           "",
					"num_def_pwd":     "FakeDEFPWD1",
					"num_defi_pmi":    "FakeDEFI1",
					"serial_num":      "test-serial-1"})},
			false,
			"",
		},
		{
			"Bad Request 400 response since serial number col is empty",
			func(r *mockstore.MockRepository) {},
			"./../../../../internal/parse/testdata/test_empty_serial_col.xlsx",
			&fleetdbapi.ServerResponse{},
			true,
			"got bad request",
		},
		{
			"Bad Request 400 response since empty serial number",
			func(r *mockstore.MockRepository) {},
			"./../../../../internal/parse/testdata/test_empty_serial.xlsx",
			&fleetdbapi.ServerResponse{},
			true,
			"got bad request",
		},
		{
			"Bad Request 400 response since empty bmc mac address",
			func(r *mockstore.MockRepository) {},
			"./../../../../internal/parse/testdata/test_empty_bmcMacAddress.xlsx",
			&fleetdbapi.ServerResponse{},
			true,
			"got bad request",
		},
		{
			"Bad Request 400 response since empty aoc mac address",
			func(r *mockstore.MockRepository) {},
			"./../../../../internal/parse/testdata/test_empty_aocMacAddress.xlsx",
			&fleetdbapi.ServerResponse{},
			true,
			"got bad request",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.mockStore != nil {
				tc.mockStore(tester.repository)
			}

			got, err := tester.client.GetBomInfoByAOCMacAddr(context.TODO(), tc.aocMacAddr)
			if tc.expectError {
				if err == nil {
					t.Errorf("GetBomInfoByAOCMacAddr(%v) expected error %v, got nil", tc.aocMacAddr, tc.expectErrorContains)
					return
				}
				if !strings.Contains(err.Error(), tc.expectErrorContains) {
					t.Errorf("GetBomInfoByAOCMacAddr(%v) expect error message %v, got %v", tc.aocMacAddr, tc.expectErrorContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("GetBomInfoByAOCMacAddr(%v) got err %v, expect nil", tc.aocMacAddr, err)
				return
			}

			if !reflect.DeepEqual(got, tc.expectResponse) {
				t.Errorf("GetBomInfoByAOCMacAddr(%v) receives %v, expects %v", tc.aocMacAddr, got, tc.expectResponse)
			}
		})
	}
}
