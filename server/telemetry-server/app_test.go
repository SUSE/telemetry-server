package main

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
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
	app           *app.App
	config        *app.Config
	router        *mux.Router
	path          string
	authToken     string
	clientReg     types.ClientRegistration
	clientRegHash types.ClientRegistrationHash
	regId         int64
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
  operational:
    driver: sqlite3
    params: %s/operational.db
  staging:
    driver: sqlite3
    params: %s/staging.db
logging:
  level: debug
auth:
  secret: VGVzdGluZ1NlY3JldAo=
`

	formattedContents := fmt.Sprintf(content, s.path, s.path, s.path)
	_, err = tmpfile.Write([]byte(formattedContents))
	require.NoError(s.T(), err)
	require.NoError(s.T(), tmpfile.Close())

	s.config = app.NewConfig(tmpfile.Name())
	err = s.config.Load()
	require.NoError(s.T(), err)

	// Initialize your app and setup a router with debug mode enabled
	s.app, s.router = InitializeApp(s.config, true)

	// setup the client entry in the clients table
	s.authToken, _ = s.app.AuthManager.CreateToken()
	s.clientReg = types.ClientRegistration{
		ClientId:   "1b504dca-bd71-424f-87f6-21eb7f5745db",
		SystemUUID: "3f97d439-5212-4688-af22-ad0559a626cb",
		Timestamp:  "2024-06-30T23:59:59.999999999Z",
	}
	s.clientRegHash = types.ClientRegistrationHash{
		Method: "sha256",
		Value:  "1a374d367946699bddce3c749ec755ce4b8859c4c9984f3c1f41460ce3bbed9c",
	}
	row := s.app.OperationalDB.Conn.QueryRow(
		`INSERT INTO clients(`+
			`clientId, `+
			`systemUUID, `+
			`clientTimestamp, `+
			`registrationDate, `+
			`authToken) `+
			`VALUES(?, ?, ?, ?, ?) `+
			`RETURNING id`,
		s.clientReg.ClientId,
		s.clientReg.SystemUUID,
		s.clientReg.Timestamp,
		"2024-07-01T00:00:00.000000000Z",
		s.authToken,
	)
	if err := row.Scan(&s.regId); err != nil {
		panic(fmt.Errorf("failed to setup test client entry in clients table: %s", err.Error()))
	}
}

func (s *AppTestSuite) TearDownTest() {
	log.Println("TeardownTest()")
	os.RemoveAll(s.path)
}

func (t *AppTestSuite) TestReportTelemetry() {
	// Test the handler wrapper.reportTelemetry
	// Simulated request handled via the router's ServeHTTP
	// Response recorded via the httptest.HttpRecorder

	body := createReportPayload()

	rr, err := postToReportTelemetryHandler(body, "", true, t)
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

func (t *AppTestSuite) TestReportTelemetryCompressedPayloadGZIP() {
	// Test the handler wrapper.reportTelemetry with compressed payload
	// Simulated request handled via the router's ServeHTTP
	// Response recorded via the httptest.HttpRecorder

	body := createReportPayload()

	//Compress payload
	cbody, err := compressedData([]byte(body), "gzip")
	assert.NoError(t.T(), err)

	rr, err := postToReportTelemetryHandler(string(cbody), "gzip", true, t)
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

func (t *AppTestSuite) TestReportTelemetryCompressedPayloadDeflate() {
	// Test the handler wrapper.reportTelemetry with compressed payload
	// Simulated request handled via the router's ServeHTTP
	// Response recorded via the httptest.HttpRecorder

	body := createReportPayload()

	//Compress payload
	cbody, err := compressedData([]byte(body), "deflate")
	assert.NoError(t.T(), err)

	rr, err := postToReportTelemetryHandler(string(cbody), "deflate", true, t)
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
	body := `{"clientRegistration":{"clientId":"%s","systemUUID":"%s", "timestamp":"%s"}}`
	formattedBody := fmt.Sprintf(body, id, "", "2024-08-01T00:00:01.000000000Z")

	rr, err := postToRegisterClientHandler(formattedBody, t)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 200, rr.Code)

	//Validate the response has these attributes
	substrings := []string{"registrationId", "authToken", "registrationDate"}
	for _, substring := range substrings {
		if !strings.Contains(rr.Body.String(), substring) {
			t.T().Errorf("String '%s' does not contain substring '%s'", rr.Body.String(), substring)
		}
	}

}

func (t *AppTestSuite) TestAuthenticateClient() {
	//Test the wrapper.autenticateClient handler

	// Create a POST request with the necessary body
	body := `{"registrationId":%d,"regHash":{"method":"%s","value":"%s"}}`
	formattedBody := fmt.Sprintf(
		body, t.regId, t.clientRegHash.Method, t.clientRegHash.Value)

	rr, err := postToAuthenticateClientHandler(formattedBody, t)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 200, rr.Code)

	//Validate the response has these attributes
	substrings := []string{"registrationId", "authToken", "registrationDate"}
	for _, substring := range substrings {
		if !strings.Contains(rr.Body.String(), substring) {
			t.T().Errorf("String '%s' does not contain substring '%s'", rr.Body.String(), substring)
		}
	}

}

func (t *AppTestSuite) TestReportTelemetryWithInvalidJSON() {
	// Create a POST request with the necessary body
	body := `{"header":{reportTimeStamp":"2024-05-29T23:45:34.871802018Z","reportClientId":1,"reportAnnotations":["abc=pqr","xyz"]},"telemetryBundles":[{"header":{"bundleId":"702ef1ed-5a38-440e-9680-357ca8d36a42","bundleTimeStamp":"2024-05-29T23:45:34.670907855Z","bundleClientId":"78b81c06-2892-4c35-b528-15db6baa0a0f","bundleCustomerId":"1234567890","bundleAnnotations":["abc=pqr","xyz"]},"telemetryDataItems":[{"header":{"telemetryId":"b016f023-77bc-4538-a82e-a1e1a2b8e9c8","telemetryTimeStamp":"2024-05-29T23:45:34.57108633Z","telemetryType":"SLE-SERVER-Test","telemetryAnnotations":["abc=pqr","xyz"]},"telemetryData":{"ItemA":1,"ItemB":"b"},"footer":{"checksum":"ichecksum"}}],"footer":{"checksum":"bchecksum"}}],"footer":{"checksum":"rchecksum"}}`

	rr, err := postToReportTelemetryHandler(body, "", true, t)
	assert.NoError(t.T(), err)

	assert.Equal(t.T(), 400, rr.Code)

}

func (t *AppTestSuite) TestReportTelemetryMissingAuth() {
	// Create a POST request with the necessary body
	body := `{"header":{reportTimeStamp":"2024-05-29T23:45:34.871802018Z","reportClientId":1,"reportAnnotations":["abc=pqr","xyz"]},"telemetryBundles":[{"header":{"bundleId":"702ef1ed-5a38-440e-9680-357ca8d36a42","bundleTimeStamp":"2024-05-29T23:45:34.670907855Z","bundleClientId":"78b81c06-2892-4c35-b528-15db6baa0a0f","bundleCustomerId":"1234567890","bundleAnnotations":["abc=pqr","xyz"]},"telemetryDataItems":[{"header":{"telemetryId":"b016f023-77bc-4538-a82e-a1e1a2b8e9c8","telemetryTimeStamp":"2024-05-29T23:45:34.57108633Z","telemetryType":"SLE-SERVER-Test","telemetryAnnotations":["abc=pqr","xyz"]},"telemetryData":{"ItemA":1,"ItemB":"b"},"footer":{"checksum":"ichecksum"}}],"footer":{"checksum":"bchecksum"}}],"footer":{"checksum":"rchecksum"}}`

	rr, err := postToReportTelemetryHandler(body, "", false, t)
	assert.NoError(t.T(), err)

	assert.Equal(t.T(), 401, rr.Code)
}

func (t *AppTestSuite) TestRegisterClientWithInvalidJSON() {
	// Create a POST request with the necessary body
	body := `{"clientRegistration":{}}`

	rr, err := postToRegisterClientHandler(body, t)
	assert.NoError(t.T(), err)

	assert.Equal(t.T(), 400, rr.Code)

}

func (t *AppTestSuite) TestAuthenticateClientWithInvalidJSON() {
	// Create a POST request with the necessary body
	body := `{"registrationId":{}}`

	rr, err := postToAuthenticateClientHandler(body, t)
	assert.NoError(t.T(), err)

	assert.Equal(t.T(), 400, rr.Code)

}

func (t *AppTestSuite) TestAuthenticateClientWithInvalidClientId() {
	// Create a POST request with the necessary body
	bodyfmt := `{"clientId":%d,"instIdHash":{"method":"%s","value":"%s"}}`
	body := fmt.Sprintf(
		bodyfmt, -1, t.clientRegHash.Method, t.clientRegHash.Value)

	rr, err := postToAuthenticateClientHandler(body, t)
	assert.NoError(t.T(), err)

	assert.Equal(t.T(), 401, rr.Code)

	wwwAuthList, ok := rr.Result().Header[http.CanonicalHeaderKey("WWW-Authenticate")]
	assert.True(t.T(), ok, "missing WWW-Authenticate header")
	assert.True(t.T(), strings.Contains(wwwAuthList[0], "Bearer"), "WWW-Authenticate should contain Bearer challenge")
	assert.True(t.T(), strings.Contains(wwwAuthList[0], `realm="suse-telemetry-service"`), "WWW-Authenticate should contain correct realm")
	assert.True(t.T(), strings.Contains(wwwAuthList[0], `scope="register"`), "WWW-Authenticate should contain correct scope")
}

func (t *AppTestSuite) TestAuthenticateClientWithUnregisteredClientId() {
	// Create a POST request with the necessary body
	bodyfmt := `{"clientId":%d,"instIdHash":{"method":"%s","value":"%s"}}`
	body := fmt.Sprintf(
		bodyfmt, t.regId+1, t.clientRegHash.Method, t.clientRegHash.Value)

	rr, err := postToAuthenticateClientHandler(body, t)
	assert.NoError(t.T(), err)

	assert.Equal(t.T(), 401, rr.Code)

	wwwAuthList, ok := rr.Result().Header[http.CanonicalHeaderKey("WWW-Authenticate")]
	assert.True(t.T(), ok, "missing WWW-Authenticate header")
	assert.True(t.T(), strings.Contains(wwwAuthList[0], "Bearer"), "WWW-Authenticate should contain Bearer challenge")
	assert.True(t.T(), strings.Contains(wwwAuthList[0], `realm="suse-telemetry-service"`), "WWW-Authenticate should contain correct realm")
	assert.True(t.T(), strings.Contains(wwwAuthList[0], `scope="register"`), "WWW-Authenticate should contain correct scope")
}

func (t *AppTestSuite) TestAuthenticateClientWithInvalidInstIdHash() {
	// Create a POST request with the necessary body
	bodyfmt := `{"clientId":%d,"instIdHash":{"method":"%s","value":"%s"}}`
	body := fmt.Sprintf(
		bodyfmt, t.regId+1, "sha512", t.clientRegHash.Value)

	rr, err := postToAuthenticateClientHandler(body, t)
	assert.NoError(t.T(), err)

	assert.Equal(t.T(), 401, rr.Code)

}

func (t *AppTestSuite) TestReportTelemetryWithEmptyPayload() {
	//Test the wrapper.reportTelemetry handler
	// Create a POST request with the necessary body
	body := `{}`
	rr, err := postToReportTelemetryHandler(body, "", true, t)
	assert.NoError(t.T(), err)

	assert.Equal(t.T(), 400, rr.Code)

}

func (t *AppTestSuite) TestReportTelemetryWithEmptyValues() {
	tests := []struct {
		name       string
		body       string
		shouldFail bool
	}{

		{"Validation with header.reportId empty value",
			`{"header":{"reportId":"","reportTimeStamp":"%s","reportClientId":"0997e7bb-ce76-4a4d-a0b7-07aeeb6ead66","reportAnnotations":["rkey1=rvalue1","rkey2"]},"telemetryBundles":[{"header":{"bundleId":"1c3f3f72-1cd3-4424-a5bf-5d1c51dde2a1","bundleTimeStamp":"%s","bundleClientId":"78b81c06-2892-4c35-b528-15db6baa0a0f","bundleCustomerId":"customer id","bundleAnnotations":["bkey1=bvalue1","bkey2"]},"telemetryDataItems":[{"header":{"telemetryId":"f4301ecc-ca03-4c31-8a3e-79b8e23e7901","telemetryTimeStamp":"%s","telemetryType":"SLE-SERVER-Test","telemetryAnnotations":["ikey1=ivalue1","ikey2"]},"telemetryData":{"ItemA":1,"ItemB":"b","ItemC":"c"},"footer":{"checksum":"ichecksum"}},{"header":{"telemetryId":"f256fdb4-22b3-462a-b8f7-9b108b49b771","telemetryTimeStamp":"%s","telemetryType":"SLE-SERVER-Test","telemetryAnnotations":["ikey1=ivalue1"]},"telemetryData":{"ItemA":1,"ItemB":"b","ItemC":
			"c"},"footer":{"checksum":"ichecksum"}}],"footer":{"checksum":"bchecksum"}}],"footer":{"checksum":"rchecksum"}}`, true},

		{"Validation with header.reportAnnotations empty list",
			`{"header":{"reportId":"fasdklfsdlfksdkflsdf2","reportTimeStamp":"%s","reportClientId":"0997e7bb-ce76-4a4d-a0b7-07aeeb6ead66","reportAnnotations":[]},"telemetryBundles":[{"header":{"bundleId":"1c3f3f72-1cd3-4424-a5bf-5d1c51dde2a2","bundleTimeStamp":"%s","bundleClientId":"78b81c06-2892-4c35-b528-15db6baa0a0f","bundleCustomerId":"customer id","bundleAnnotations":["bkey1=bvalue1","bkey2"]},"telemetryDataItems":[{"header":{"telemetryId":"f4301ecc-ca03-4c31-8a3e-79b8e23e7902","telemetryTimeStamp":"%s","telemetryType":"SLE-SERVER-Test","telemetryAnnotations":["ikey1=ivalue1","ikey2"]},"telemetryData":{"ItemA":1,"ItemB":"b","ItemC":"c"},"footer":{"checksum":"ichecksum"}},{"header":{"telemetryId":"f256fdb4-22b3-462a-b8f7-9b108b49b772","telemetryTimeStamp":"%s","telemetryType":"SLE-SERVER-Test","telemetryAnnotations":["ikey1=ivalue1"]},"telemetryData":{"ItemA":1,"ItemB":"b","ItemC":
			"c"},"footer":{"checksum":"ichecksum"}}],"footer":{"checksum":"bchecksum"}}],"footer":{"checksum":"rchecksum"}}`, false},

		{"Validation with no header.reportAnnotations attribute",
			`{"header":{"reportId":"fasdklfsdlfkssdfadkflsdf3","reportTimeStamp":"%s","reportClientId":"0997e7bb-ce76-4a4d-a0b7-07aeeb6ead66"},"telemetryBundles":[{"header":{"bundleId":"1c3f3f72-1cd3-4424-a5bf-5d1c51dde2a3","bundleTimeStamp":"%s","bundleClientId":"78b81c06-2892-4c35-b528-15db6baa0a0f","bundleCustomerId":"customer id","bundleAnnotations":["bkey1=bvalue1","bkey2"]},"telemetryDataItems":[{"header":{"telemetryId":"f4301ecc-ca03-4c31-8a3e-79b8e23e7903","telemetryTimeStamp":"%s","telemetryType":"SLE-SERVER-Test","telemetryAnnotations":["ikey1=ivalue1","ikey2"]},"telemetryData":{"ItemA":1,"ItemB":"b","ItemC":"c"},"footer":{"checksum":"ichecksum"}},{"header":{"telemetryId":"f256fdb4-22b3-462a-b8f7-9b108b49b773","telemetryTimeStamp":"%s","telemetryType":"SLE-SERVER-Test","telemetryAnnotations":["ikey1=ivalue1"]},"telemetryData":{"ItemA":1,"ItemB":"b","ItemC":
			"c"},"footer":{"checksum":"ichecksum"}}],"footer":{"checksum":"bchecksum"}}],"footer":{"checksum":"rchecksum"}}`, false},

		{"Validation with no telemetryBundles attribute",
			`{"header":{"reportId":"fasdklfsdlfkssdfadkflsdf4","reportTimeStamp":"%s","reportClientId":"0997e7bb-ce76-4a4d-a0b7-07aeeb6ead66"},"footer":{"checksum":"rchecksum"}}`, true},

		{"Validation with empty telemetryBundles list",
			`{"header":{"reportId":"fasdklfsdlfkssdfadkflsdf5","reportTimeStamp":"%s","reportClientId":"0997e7bb-ce76-4a4d-a0b7-07aeeb6ead66"},"telemetryBundles":[],"footer":{"checksum":"rchecksum"}}`, true},

		{"Validation with bundleId empty value",
			`{"header":{"reportId":"fasdklfsdlfkssdfadkflsdf6","reportTimeStamp":"%s","reportClientId":"0997e7bb-ce76-4a4d-a0b7-07aeeb6ead66","reportAnnotations":["rkey1=rvalue1","rkey2"]},"telemetryBundles":[{"header":{"bundleId":"","bundleTimeStamp":"%s","bundleClientId":"78b81c06-2892-4c35-b528-15db6baa0a0f","bundleCustomerId":"customer id","bundleAnnotations":["bkey1=bvalue1","bkey2"]},"telemetryDataItems":[{"header":{"telemetryId":"f4301ecc-ca03-4c31-8a3e-79b8e23e7904","telemetryTimeStamp":"%s","telemetryType":"SLE-SERVER-Test","telemetryAnnotations":["ikey1=ivalue1","ikey2"]},"telemetryData":{"ItemA":1,"ItemB":"b","ItemC":"c"},"footer":{"checksum":"ichecksum"}},{"header":{"telemetryId":"f256fdb4-22b3-462a-b8f7-9b108b49b774","telemetryTimeStamp":"%s","telemetryType":"SLE-SERVER-Test","telemetryAnnotations":["ikey1=ivalue1"]},"telemetryData":{"ItemA":1,"ItemB":"b","ItemC":
			"c"},"footer":{"checksum":"ichecksum"}}],"footer":{"checksum":"bchecksum"}}],"footer":{"checksum":"rchecksum"}}`, true},

		{"Validation with bundleAnnotations empty list",
			`{"header":{"reportId":"fasdklfsdlfkssdfadkflsdf6","reportTimeStamp":"%s","reportClientId":"0997e7bb-ce76-4a4d-a0b7-07aeeb6ead66","reportAnnotations":["rkey1=rvalue1","rkey2"]},"telemetryBundles":[{"header":{"bundleId":"1c3f3f72-1cd3-4424-a5bf-5d1c51dde2b1","bundleTimeStamp":"%s","bundleClientId":"78b81c06-2892-4c35-b528-15db6baa0a0f","bundleCustomerId":"customer id","bundleAnnotations":[]},"telemetryDataItems":[{"header":{"telemetryId":"f4301ecc-ca03-4c31-8a3e-79b8e23e7904","telemetryTimeStamp":"%s","telemetryType":"SLE-SERVER-Test","telemetryAnnotations":["ikey1=ivalue1","ikey2"]},"telemetryData":{"ItemA":1,"ItemB":"b","ItemC":"c"},"footer":{"checksum":"ichecksum"}},{"header":{"telemetryId":"f256fdb4-22b3-462a-b8f7-9b108b49b774","telemetryTimeStamp":"%s","telemetryType":"SLE-SERVER-Test","telemetryAnnotations":["ikey1=ivalue1"]},"telemetryData":{"ItemA":1,"ItemB":"b","ItemC":
			"c"},"footer":{"checksum":"ichecksum"}}],"footer":{"checksum":"bchecksum"}}],"footer":{"checksum":"rchecksum"}}`, false},

		{"Validation with empty telemetryAnnotations",
			`{"header":{"reportId":"fasdklfsdlfkssdfadkflsde8","reportTimeStamp":"%s","reportClientId":"0997e7bb-ce76-4a4d-a0b7-07aeeb6ead66","reportAnnotations":["rkey1=rvalue1","rkey2"]},"telemetryBundles":[{"header":{"bundleId":"1c3f3f72-1cd3-4424-a5bf-5d1c51dde2a1","bundleTimeStamp":"%s","bundleClientId":"78b81c06-2892-4c35-b528-15db6baa0a0f","bundleCustomerId":"customer id","bundleAnnotations":["bkey1=bvalue1","bkey2"]},"telemetryDataItems":[{"header":{"telemetryId":"f4301ecc-ca03-4c31-8a3e-79b8e23e7901","telemetryTimeStamp":"%s","telemetryType":"SLE-SERVER-Test","telemetryAnnotations":[]},"telemetryData":{"ItemA":1,"ItemB":"b","ItemC":"c"},"footer":{"checksum":"ichecksum"}},{"header":{"telemetryId":"f256fdb4-22b3-462a-b8f7-9b108b49b771","telemetryTimeStamp":"%s","telemetryType":"SLE-SERVER-Test","telemetryAnnotations":[]},"telemetryData":{"ItemA":1,"ItemB":"b","ItemC":"c"},"footer":{"checksum":"ichecksum"}}],"footer":{"checksum":"bchecksum"}}],"footer":{"checksum":"rchecksum"}}`, false},

		{"Validation with no telemetryAnnotations attribute",
			`{"header":{"reportId":"fasdklfsdlfkssdfadkflsde9","reportTimeStamp":"%s","reportClientId":"0997e7bb-ce76-4a4d-a0b7-07aeeb6ead66","reportAnnotations":["rkey1=rvalue1","rkey2"]},"telemetryBundles":[{"header":{"bundleId":"1c3f3f72-1cd3-4424-a5bf-5d1c51dde2a1","bundleTimeStamp":"%s","bundleClientId":"78b81c06-2892-4c35-b528-15db6baa0a0f","bundleCustomerId":"customer id","bundleAnnotations":["bkey1=bvalue1","bkey2"]},"telemetryDataItems":[{"header":{"telemetryId":"f4301ecc-ca03-4c31-8a3e-79b8e23e7901","telemetryTimeStamp":"%s","telemetryType":"SLE-SERVER-Test"},"telemetryData":{"ItemA":1,"ItemB":"b","ItemC":"c"},"footer":{"checksum":"ichecksum"}},{"header":{"telemetryId":"f256fdb4-22b3-462a-b8f7-9b108b49b771","telemetryTimeStamp":"%s","telemetryType":"SLE-SERVER-Test"},"telemetryData":{"ItemA":1,"ItemB":"b","ItemC":"c"},"footer":{"checksum":"ichecksum"}}],"footer":{"checksum":"bchecksum"}}],"footer":{"checksum":"rchecksum"}}`, false},
	}

	for _, tt := range tests {
		t.Run("Report Telemetry "+tt.name, func() {
			tm := types.Now().String()
			formattedBody := fmt.Sprintf(tt.body, tm, tm, tm, tm)

			//Test the wrapper.reportTelemetry handler
			// Create a POST request with the necessary body

			rr, err := postToReportTelemetryHandler(formattedBody, "", true, t)
			assert.NoError(t.T(), err)

			if tt.shouldFail {
				assert.Equal(t.T(), http.StatusBadRequest, rr.Code)
			} else {
				assert.Equal(t.T(), http.StatusOK, rr.Code)
			}

		})
	}
}

func (t *AppTestSuite) TestRegisterClientWithEmptyJSON() {
	// Create a POST request with the necessary body
	body := `{}`
	rr, err := postToRegisterClientHandler(body, t)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 400, rr.Code)
}

func (t *AppTestSuite) TestAuthenticateClientWithEmptyJSON() {
	// Create a POST request with the necessary body
	body := `{}`
	rr, err := postToAuthenticateClientHandler(body, t)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 401, rr.Code)

	wwwAuthList, ok := rr.Result().Header[http.CanonicalHeaderKey("WWW-Authenticate")]
	assert.True(t.T(), ok, "missing WWW-Authenticate header")
	assert.True(t.T(), strings.Contains(wwwAuthList[0], "Bearer"), "WWW-Authenticate should contain Bearer challenge")
	assert.True(t.T(), strings.Contains(wwwAuthList[0], `realm="suse-telemetry-service"`), "WWW-Authenticate should contain correct realm")
	assert.True(t.T(), strings.Contains(wwwAuthList[0], `scope="register"`), "WWW-Authenticate should contain correct scope")
}

func postToReportTelemetryHandler(body string, compression string, auth bool, t *AppTestSuite) (*httptest.ResponseRecorder, error) {
	req, err := http.NewRequest("POST", "/telemetry/report", strings.NewReader(body))
	assert.NoError(t.T(), err)

	switch compression {
	case "gzip":
		req.Header.Set("Content-Encoding", "gzip")
	case "deflate":
		req.Header.Set("Content-Encoding", "deflate")
	default:
		//No compression
	}

	req.Header.Set("Content-Type", "application/json")
	if auth {
		req.Header.Set("Authorization", "Bearer "+t.authToken)
	}
	req.Header.Set("X-Telemetry-Registration-Id", fmt.Sprintf("%d", t.regId))

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

func postToAuthenticateClientHandler(body string, t *AppTestSuite) (*httptest.ResponseRecorder, error) {
	req, err := http.NewRequest("POST", "/telemetry/authenticate", strings.NewReader(body))
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

func createReportPayload() (reportPayload string) {
	// Create 2 dataitems
	telemetryType := types.TelemetryType("SLE-SERVER-Test")
	itags1 := types.Tags{types.Tag("ikey1=ivalue1"), types.Tag("ikey2")}
	itags2 := types.Tags{types.Tag("ikey1=ivalue1")}
	payload := types.NewTelemetryBlob([]byte(`{
		"ItemA": 1,
		"ItemB": "b",
		"ItemC": "c"
	}`))

	item1 := telemetrylib.NewTelemetryDataItem(telemetryType, itags1, payload)
	item2 := telemetrylib.NewTelemetryDataItem(telemetryType, itags2, payload)

	client_id := uuid.New().String()

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

func compressedData(data []byte, alg string) (b []byte, err error) {
	switch alg {
	case "gzip":
		return compress(data, func(w io.Writer) io.WriteCloser {
			return gzip.NewWriter(w)
		})
	case "deflate":
		return compress(data, func(w io.Writer) io.WriteCloser {
			return zlib.NewWriter(w)
		})
	default:
		//default compression gzip
		return compress(data, func(w io.Writer) io.WriteCloser {
			return gzip.NewWriter(w)
		})
	}
}

func compress(data []byte, writerFunc func(io.Writer) io.WriteCloser) ([]byte, error) {
	var buf bytes.Buffer
	writer := writerFunc(&buf)
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (t *AppTestSuite) TestHealthCheckHandler() {
	//Test the wrapper.healthCheck handler
	req, err := http.NewRequest("GET", "/healthz", nil)
	assert.NoError(t.T(), err)
	req.Header.Set("Content-Type", "application/json")

	// Record the response
	rr := httptest.NewRecorder()

	t.router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t.T(), http.StatusOK, rr.Code)

}
