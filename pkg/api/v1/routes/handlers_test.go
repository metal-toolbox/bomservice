package routes

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/metal-toolbox/hollow-bomservice/internal/store"
	mockstore "github.com/metal-toolbox/hollow-bomservice/internal/store/mock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	sservice "go.hollow.sh/serverservice/pkg/api/v1"
	"go.hollow.sh/toolbox/events"
)

// TODO: add robust test cases to test handlers.

var testDatapath = "./../../../../internal/parse/testdata"

func mockserver(t *testing.T, logger *logrus.Logger, repository store.Repository, stream events.Stream) (*gin.Engine, error) {
	t.Helper()

	gin.SetMode(gin.ReleaseMode)
	g := gin.New()
	g.Use(gin.Recovery())

	options := []Option{
		WithLogger(logger),
		WithStore(repository),
	}

	v1Router, err := NewRoutes(options...)
	if err != nil {
		return nil, err
	}

	v1Router.Routes(g.Group("/api/v1"))

	g.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"message": "invalid request - route not found"})
	})

	return g, nil
}

func TestUploadXlsxFile(t *testing.T) {
	// mock repository
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repository := mockstore.NewMockRepository(ctrl)
	server, err := mockserver(t, logrus.New(), repository, nil)
	if err != nil {
		t.Fatal(err)
	}

	validBoms :=
		[]sservice.Bom{
			{
				SerialNum:     "test-serial-1",
				AocMacAddress: "FakeAOC1,FakeAOC2",
				BmcMacAddress: "FakeMac1,FakeMac2",
				NumDefiPmi:    "FakeDEFI1",
				NumDefPWD:     "FakeDEFPWD1",
			},
			{
				SerialNum:     "test-serial-2",
				AocMacAddress: "FakeAOC3,FakeAOC4",
				BmcMacAddress: "FakeMac3,FakeMac4",
				NumDefiPmi:    "FakeDEFI2",
				NumDefPWD:     "FakeDEFPWD2",
			},
		}

	testcases := []struct {
		name           string
		fileName       string
		mockStore      func(r *mockstore.MockRepository)
		assertResponse func(t *testing.T, r *httptest.ResponseRecorder)
	}{
		{
			"valid file with 2 boms",
			"test_valid_multiple_boms.xlsx",
			func(r *mockstore.MockRepository) {
				r.EXPECT().
					BillOfMaterialsBatchUpload(
						gomock.Any(),
						gomock.Eq(validBoms),
					).
					Return(nil, nil).
					Times(1)
			},
			func(t *testing.T, r *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, r.Code)
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.mockStore != nil {
				tc.mockStore(repository)
			}

			filePath := fmt.Sprintf("%v/%v", testDatapath, tc.fileName)
			file, err := os.Open(filePath)
			if err != nil {
				t.Fatalf("os.Open(%v) failed to open file %v\n", filePath, err)
			}
			reader := bufio.NewReader(file)
			request, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/api/v1/bomservice/upload-xlsx-file", reader)
			if err != nil {
				t.Fatal(err)
			}

			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, request)

			tc.assertResponse(t, recorder)
		})
	}
}

func TestGetBomInfoByAocMacAddr(t *testing.T) {
	// mock repository
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repository := mockstore.NewMockRepository(ctrl)
	server, err := mockserver(t, logrus.New(), repository, nil)
	if err != nil {
		t.Fatal(err)
	}

	validBom := &sservice.Bom{
		SerialNum:     "test-serial-1",
		AocMacAddress: "FakeAOC1,FakeAOC2",
		BmcMacAddress: "FakeMac1,FakeMac2",
		NumDefiPmi:    "FakeDEFI1",
		NumDefPWD:     "FakeDEFPWD1",
	}

	testcases := []struct {
		name           string
		aocMacAddr     string
		mockStore      func(r *mockstore.MockRepository)
		assertResponse func(t *testing.T, r *httptest.ResponseRecorder)
	}{
		{
			"valid aoc mac address and it is in the DB",
			"test-serial-1",
			func(r *mockstore.MockRepository) {
				r.EXPECT().
					GetBomInfoByAOCMacAddr(
						gomock.Any(),
						gomock.Eq("test-serial-1"),
					).
					Return(validBom, &sservice.ServerResponse{
						Record: validBom,
					}, nil).
					Times(1)
			},
			func(t *testing.T, r *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, r.Code)
				var resp sservice.ServerResponse
				err := json.Unmarshal(r.Body.Bytes(), &resp)
				assert.NoError(t, err, "malformed response body")
				jsonStr, err := json.Marshal(resp.Record)
				assert.NoError(t, err, "malformed ServerResponse record")
				var bom sservice.Bom
				err = json.Unmarshal(jsonStr, &bom)
				assert.NoError(t, err, "malformed ServerResponse record")
				// not using assert.True in order to dump the value of 2 objects
				if !reflect.DeepEqual(bom, *validBom) {
					t.Errorf("HTTP receives %v, expects %v", resp.Record, validBom)
				}
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.mockStore != nil {
				tc.mockStore(repository)
			}
			url := fmt.Sprintf("%v/%v", "/api/v1/bomservice/aoc-mac-address", tc.aocMacAddr)
			request, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, url, nil)
			if err != nil {
				t.Fatal(err)
			}

			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, request)

			tc.assertResponse(t, recorder)
		})
	}
}
