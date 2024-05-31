package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/SUSE/telemetry/pkg/types"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/SUSE/telemetry-server/app"

	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
)

type AppTestSuite struct {
	suite.Suite
	app    *app.App
	config *app.Config
	router *mux.Router
	path   string
}

// run before each test
func (s *AppTestSuite) SetupTest() {
	log.Println("SetupTest()")
	//server configuration
	var err error
	s.path, err = os.MkdirTemp("", "telemetry-server-test-")
	require.NoError(s.T(), err)
	tmpfile, err := os.CreateTemp(s.path, "*.yaml")
	require.NoError(s.T(), err)

	content := `
---
api:
  host: localhost
  port: 9999
dbs:
  telemetry:
    driver: sqlite3
    params: %s/telemetry.db
  staging:
    driver: sqlite3
    params: %s/staging.db
`

	formattedContents := fmt.Sprintf(content, s.path, s.path)
	_, err = tmpfile.Write([]byte(formattedContents))
	require.NoError(s.T(), err)
	require.NoError(s.T(), tmpfile.Close())

	s.config = app.NewConfig(tmpfile.Name())
	err = s.config.Load()
	require.NoError(s.T(), err)

	// Initialize your app and setup a router
	s.app, s.router = InitializeApp(s.config)

}

func (s *AppTestSuite) TearDownTest() {
	log.Println("TeardownTest()")
	os.RemoveAll(s.path)
}

func (t *AppTestSuite) TestReportTelemetry() {
	// Test the handler wrapper.reportTelemetry
	// Simulated request handled via the router's ServeHTTP
	// Response recorded via the httptest.HttpRecorder

	body := createReportPayload(t.T())

	rr, err := postToReportTelemetryHandler(body, t)
	assert.NoError(t.T(), err)

	assert.Equal(t.T(), 200, rr.Code)

	//Validate the response has these attributes
	substrings := []string{"processingId", "processedAt"}
	for _, substring := range substrings {
		if !strings.Contains(rr.Body.String(), substring) {
			t.T().Errorf("String '%s' does not contain substring '%s'", rr.Body.String(), substring)
		}
	}

}

func (t *AppTestSuite) TestRegisterClient() {
	//Test the wrapper.registerClient handler

	// Create a POST request with the necessary body
	id := uuid.New().String()
	body := `{"clientInstanceId":"%s"}`
	formattedBody := fmt.Sprintf(body, id)

	rr, err := postToRegisterClientHandler(formattedBody, t)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 200, rr.Code)

	//Validate the response has these attributes
	substrings := []string{"clientId", "authToken", "issueDate"}
	for _, substring := range substrings {
		if !strings.Contains(rr.Body.String(), substring) {
			t.T().Errorf("String '%s' does not contain substring '%s'", rr.Body.String(), substring)
		}
	}

}

func (t *AppTestSuite) TestReportTelemetryWithInvalidJSON() {
	// Create a POST request with the necessary body
	body := `{"header":{reportTimeStamp":"2024-05-29T23:45:34.871802018Z","reportClientId":1,"reportAnnotations":["abc=pqr","xyz"]},"telemetryBundles":[{"header":{"bundleId":"702ef1ed-5a38-440e-9680-357ca8d36a42","bundleTimeStamp":"2024-05-29T23:45:34.670907855Z","bundleClientId":1,"buncleCustomerId":"1234567890","bundleAnnotations":["abc=pqr","xyz"]},"telemetryDataItems":[{"header":{"telemetryId":"b016f023-77bc-4538-a82e-a1e1a2b8e9c8","telemetryTimeStamp":"2024-05-29T23:45:34.57108633Z","telemetryType":"SLE-SERVER-Test","telemetryAnnotations":["abc=pqr","xyz"]},"telemetryData":{"ItemA":1,"ItemB":"b"},"footer":{"checksum":"ichecksum"}}],"footer":{"checksum":"bchecksum"}}],"footer":{"checksum":"rchecksum"}}`

	rr, err := postToReportTelemetryHandler(body, t)
	assert.NoError(t.T(), err)

	assert.Equal(t.T(), 400, rr.Code)

}

func (t *AppTestSuite) TestRegisterClientWithInvalidJSON() {
	// Create a POST request with the necessary body
	body := `{"clientInstanceId":}`

	rr, err := postToRegisterClientHandler(body, t)
	assert.NoError(t.T(), err)

	assert.Equal(t.T(), 400, rr.Code)

}

func (t *AppTestSuite) TestReportTelemetryWithEmptyPayload() {
	//Test the wrapper.reportTelemetry handler

	t.T().Skip("Skipping this test for now")

	// Create a POST request with the necessary body
	body := `{}`
	rr, err := postToReportTelemetryHandler(body, t)
	assert.NoError(t.T(), err)

	assert.Equal(t.T(), 400, rr.Code)

}

func (t *AppTestSuite) TestRegisterClientWithEmptyJSON() {
	// Create a POST request with the necessary body
	body := `{}`
	rr, err := postToRegisterClientHandler(body, t)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 400, rr.Code)
}

func postToReportTelemetryHandler(body string, t *AppTestSuite) (*httptest.ResponseRecorder, error) {
	req, err := http.NewRequest("POST", "/telemetry/report", strings.NewReader(body))
	assert.NoError(t.T(), err)
	req.Header.Set("Content-Type", "application/json")

	// Record the response
	rr := httptest.NewRecorder()

	t.router.ServeHTTP(rr, req)

	return rr, err

}

func postToRegisterClientHandler(body string, t *AppTestSuite) (*httptest.ResponseRecorder, error) {
	req, err := http.NewRequest("POST", "/telemetry/register", strings.NewReader(body))
	assert.NoError(t.T(), err)
	req.Header.Set("Content-Type", "application/json")

	// Record the response
	rr := httptest.NewRecorder()

	t.router.ServeHTTP(rr, req)

	return rr, err

}

func TestAppTestSuite(t *testing.T) {
	suite.Run(t, new(AppTestSuite))
}

func createReportPayload(t *testing.T) (reportPayload string) {
	// Create 2 dataitems
	telemetryType := types.TelemetryType("SLE-SERVER-Test")
	itags1 := types.Tags{types.Tag("ikey1=ivalue1"), types.Tag("ikey2")}
	itags2 := types.Tags{types.Tag("ikey1=ivalue1")}
	payload := `
			{
				"ItemA": 1,
				"ItemB": "b",
				"ItemC": "c"
			}
			`

	item1, err := telemetrylib.NewTelemetryDataItem(telemetryType, itags1, []byte(payload))
	assert.NoError(t, err)
	item2, err := telemetrylib.NewTelemetryDataItem(telemetryType, itags2, []byte(payload))
	assert.NoError(t, err)

	client_id := int64(12345)

	// Create 1 bundle
	btags1 := types.Tags{types.Tag("bkey1=bvalue1"), types.Tag("bkey2")}
	bundle1 := telemetrylib.NewTelemetryBundle(client_id, "customer id", btags1)

	// add the two items to the bundle
	bundle1.TelemetryDataItems = append(bundle1.TelemetryDataItems, *item1)
	bundle1.TelemetryDataItems = append(bundle1.TelemetryDataItems, *item2)

	// Create 1 report
	rtags1 := types.Tags{types.Tag("rkey1=rvalue1"), types.Tag("rkey2")}
	report1 := telemetrylib.NewTelemetryReport(client_id, rtags1)

	report1.TelemetryBundles = append(report1.TelemetryBundles, *bundle1)

	jsonData, _ := json.Marshal(report1)
	reportPayload = string(jsonData)

	return
}
