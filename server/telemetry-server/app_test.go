package main

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/SUSE/telemetry-server/app"
	"github.com/SUSE/telemetry-server/app/config"
	"github.com/SUSE/telemetry/pkg/restapi"
	"github.com/SUSE/telemetry/pkg/types"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
)

type AppTestSuite struct {
	suite.Suite
	app           *app.App
	config        *config.Config
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
logging:
  level: debug
auth:
  secret: VGVzdGluZ1NlY3JldAo=
`

	formattedContents := fmt.Sprintf(content, s.path, s.path)
	_, err = tmpfile.Write([]byte(formattedContents))
	require.NoError(s.T(), err)
	require.NoError(s.T(), tmpfile.Close())

	s.config = config.NewConfig(tmpfile.Name())
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
	row := s.app.OperationalDB.Conn().DB().QueryRow(
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

	body, err := createReportPayload("TestCustomer")
	t.NoError(err, "creating a report payload should succeed")

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

func (t *AppTestSuite) countCustomerIdEntries(customerId string) (count int, err error) {
	row := t.app.TelemetryDB.Conn().DB().QueryRow(
		`SELECT COUNT(id) from customers WHERE customerId = '` + customerId + `'`,
	)
	err = row.Scan(&count)
	if err != nil {
		if err != sql.ErrNoRows {
			return
		}
		err = nil
	}

	return
}

func (t *AppTestSuite) TestReportTelemetryAnonymously() {
	// Test that a client id that is empty, or is a case insensitive
	// match for anonymous maps to an ANONYMOUS customerId entry

	tests := []struct {
		name               string
		customerId         string
		expectedCustomerId string
	}{
		{
			name:               "Empty customerId",
			customerId:         "",
			expectedCustomerId: app.ANONYMOUS_CUSTOMER_ID,
		},
		{
			name:               "Whitespace customerId",
			customerId:         "    ",
			expectedCustomerId: app.ANONYMOUS_CUSTOMER_ID,
		},
		{
			name:               "lowercase anonymous customerId",
			customerId:         "anonymous",
			expectedCustomerId: app.ANONYMOUS_CUSTOMER_ID,
		},
		{
			name:               "mixed case anonymous customerId",
			customerId:         "anoNymoUs",
			expectedCustomerId: app.ANONYMOUS_CUSTOMER_ID,
		},
		{
			name:               "lowercase anonymous customerId with trailing whitespace",
			customerId:         "anonymous  ",
			expectedCustomerId: app.ANONYMOUS_CUSTOMER_ID,
		},
		{
			name:               "lowercase anonymous customerId with leading whitespace",
			customerId:         "  anonymous",
			expectedCustomerId: app.ANONYMOUS_CUSTOMER_ID,
		},
		{
			name:               "uppercase anonymous customerId",
			customerId:         app.ANONYMOUS_CUSTOMER_ID,
			expectedCustomerId: app.ANONYMOUS_CUSTOMER_ID,
		},
		{
			name:               "uppercase anonymous customerId with trailing whitespace",
			customerId:         app.ANONYMOUS_CUSTOMER_ID + "  ",
			expectedCustomerId: app.ANONYMOUS_CUSTOMER_ID,
		},
		{
			name:               "uppercase anonymous customerId with leading whitespace",
			customerId:         "  " + app.ANONYMOUS_CUSTOMER_ID,
			expectedCustomerId: app.ANONYMOUS_CUSTOMER_ID,
		},
	}

	for _, tt := range tests {
		t.Run("Report Telemetry with "+tt.name, func() {
			// create a payload with an empty customer id
			body, err := createReportPayload(tt.customerId)
			t.NoError(err, "creating a report payload should succeed")

			rr, err := postToReportTelemetryHandler(body, "", true, t)
			t.NoError(err, "posting telemetry should succeed")
			t.Equal(200, rr.Code)

			// only check for customer id if different than expected customer id
			if tt.customerId != tt.expectedCustomerId {
				// count number of customer id entries
				custCount, err := t.countCustomerIdEntries(tt.customerId)
				t.NoError(err, "select count statement should have succeeded")
				t.Equal(0, custCount, "a customer id entry should not have been created")
			}

			// count number of expected customer id entries
			actualCount, err := t.countCustomerIdEntries(tt.expectedCustomerId)
			t.NoError(err, "select count statement should have succeeded")
			t.Equal(1, actualCount, "expected customer id entry should have been created")

			// validate the response has expected attributes
			substrings := []string{"processingId", "processedAt"}
			for _, substring := range substrings {
				t.Contains(rr.Body.String(), substring, "String %q does not contain substring %q", rr.Body.String(), substring)
			}
		})
	}
}

func (t *AppTestSuite) TestReportTelemetryCompressedPayloadGZIP() {
	// Test the handler wrapper.reportTelemetry with compressed payload
	// Simulated request handled via the router's ServeHTTP
	// Response recorded via the httptest.HttpRecorder

	body, err := createReportPayload("TestCustomer")
	t.NoError(err, "creating a report payload should succeed")

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

	body, err := createReportPayload("TestCustomer")
	t.NoError(err, "creating a report payload should succeed")

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

type clientTestReg struct {
	Name         string
	ClientId     string
	SystemUUID   string
	Timestamp    string
	RegId        int64
	DbClientId   string
	DbSystemUUID string
	DbTimestamp  string
}

func newClientTestReg(name string) *clientTestReg {
	return &clientTestReg{
		Name:       name,
		ClientId:   uuid.New().String(),
		SystemUUID: uuid.New().String(),
		Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func (c *clientTestReg) ReqBody() string {
	return fmt.Sprintf(
		`{"clientRegistration":{"clientId":"%s","systemUUID":"%s", "timestamp":"%s"}}`,
		c.ClientId,
		c.SystemUUID,
		c.Timestamp)
}

func (t *AppTestSuite) TestRegisterClient() {
	//Test the wrapper.registerClient handler

	// create test clients
	clients := []*clientTestReg{
		newClientTestReg("one"),
		newClientTestReg("two"),
		newClientTestReg("oneagain"),
	}

	// make the 3rd client have the same clientId as the first
	clients[2].ClientId = clients[0].ClientId

	// verify that first two client clientIds are unique and the 3rd client's clientId matches the first
	t.NotEqual(clients[0].ClientId, clients[1].ClientId, "clients %q and %q should have different clientIds", clients[0].Name, clients[1].Name)
	t.NotEqual(clients[1].ClientId, clients[2].ClientId, "clients %q and %q should have different clientIds", clients[1].Name, clients[2].Name)
	t.Equal(clients[0].ClientId, clients[2].ClientId, "clients %q and %q should have the same clientId", clients[0].Name, clients[2].Name)

	// verify that each client clientId is unique
	t.NotEqual(clients[0].SystemUUID, clients[1].SystemUUID, "clients %q and %q should have different systemUUIDs", clients[0].Name, clients[1].Name)
	t.NotEqual(clients[1].SystemUUID, clients[2].SystemUUID, "clients %q and %q should have different systemUUIDs", clients[1].Name, clients[2].Name)
	t.NotEqual(clients[2].SystemUUID, clients[0].SystemUUID, "clients %q and %q should have different systemUUIDs", clients[2].Name, clients[0].Name)

	// verify that each client clientId is unique
	t.NotEqual(clients[0].Timestamp, clients[1].Timestamp, "clients %q and %q should have different timestamps", clients[0].Name, clients[1].Name)
	t.NotEqual(clients[1].Timestamp, clients[2].Timestamp, "clients %q and %q should have different timestamps", clients[1].Name, clients[2].Name)
	t.NotEqual(clients[2].Timestamp, clients[0].Timestamp, "clients %q and %q should have different timestamps", clients[2].Name, clients[0].Name)

	// register the defined clients and validate they registered correctly
	for _, client := range clients {
		// register the client
		rr, err := postToRegisterClientHandler(client.ReqBody(), t)
		t.NoError(err, "client %q /register failed", client.Name)
		t.Equal(http.StatusOK, rr.Code, "client %q status code not StatusOK", client.Name)

		// validate the response has these JSON attributes
		substrings := []string{`"registrationId"`, `"authToken"`, `"registrationDate"`}
		for _, substring := range substrings {
			if !strings.Contains(rr.Body.String(), substring) {
				t.T().Errorf("client %q reg resp %q does not contain substring %q", client.Name, rr.Body.String(), substring)
			}
		}

		// determine the client's registrationId
		var creds restapi.ClientRegistrationResponse
		err = json.Unmarshal(rr.Body.Bytes(), &creds)
		t.Require().NoError(err, "client %q reg resp json.Unmarshal() failed", client.Name)
		t.NotZero(creds.RegistrationId, "client %q reg resp registrationId should be non-zero", client.Name)

		// record the client registrationId
		client.RegId = creds.RegistrationId
	}

	// verify that each client regId is unique
	t.NotEqual(clients[0].RegId, clients[1].RegId, "clients %q and %q should have different regIds", clients[0].Name, clients[1].Name)
	t.NotEqual(clients[1].RegId, clients[2].RegId, "clients %q and %q should have different regIds", clients[1].Name, clients[2].Name)
	t.NotEqual(clients[2].RegId, clients[0].RegId, "clients %q and %q should have different regIds", clients[2].Name, clients[0].Name)

	for _, client := range clients {
		// retrieve the client registration data from the DB
		row := t.app.OperationalDB.Conn().DB().QueryRow(
			`SELECT `+
				`clientId, `+
				`systemUUID, `+
				`clientTimestamp `+
				`FROM clients WHERE id = ?`,
			client.RegId,
		)
		err := row.Scan(
			&client.DbClientId,
			&client.DbSystemUUID,
			&client.DbTimestamp,
		)
		t.Require().NoError(err, "client %q DB clients row.Scan() failed", client.Name)

		// verify that the db registration data matches submitted registration data
		t.Equal(client.ClientId, client.DbClientId, "client %q request clientId should match DB clientId", client.Name)
		t.Equal(client.SystemUUID, client.DbSystemUUID, "client %q request systemUUID should match DB systemUUID", client.Name)
		t.Equal(client.Timestamp, client.DbTimestamp, "client %q request timestamp should match DB timestamp", client.Name)
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

func copyReport(orig *telemetrylib.TelemetryReport) *telemetrylib.TelemetryReport {
	content, _ := json.Marshal(orig)
	report := new(telemetrylib.TelemetryReport)
	_ = json.Unmarshal(content, report)
	return report
}

func (t *AppTestSuite) TestReportTelemetryWithEmptyValues() {
	clientId := uuid.NewString()

	// create a report
	rtags := types.Tags{
		types.Tag("rkey1=rvalue1"),
		types.Tag("rkey2"),
	}
	report, err := telemetrylib.NewTelemetryReport(clientId, rtags)
	t.Require().NoError(err, "should be able to create a report")

	// create a bundle
	btags := types.Tags{
		types.Tag("bkey1=rvalue1"),
		types.Tag("bkey2"),
	}
	bundle, err := telemetrylib.NewTelemetryBundle(clientId, "SomeCustomer", btags)
	t.Require().NoError(err, "should be able to create a bundle")

	// create two items
	itags1 := types.Tags{
		types.Tag("ikey1=rvalue1"),
	}
	itags2 := types.Tags{
		types.Tag("ikey2"),
	}
	item1, err := telemetrylib.NewTelemetryDataItem(
		"TEL-TYPE-1",
		itags1,
		types.NewTelemetryBlob([]byte(`{"key1": "value1", "key2": "value2"}`)),
	)
	t.Require().NoError(err, "should be able to create an item")

	item2, err := telemetrylib.NewTelemetryDataItem(
		"TEL-TYPE-2",
		itags2,
		types.NewTelemetryBlob([]byte(`{"key3": "value3", "key4": "value4"}`)),
	)
	t.Require().NoError(err, "should be able to create an item")

	// add items to bundle
	bundle.TelemetryDataItems = append(
		bundle.TelemetryDataItems,
		*item1,
		*item2,
	)
	t.Require().NoError(bundle.UpdateChecksum(), "should be able to update bundle checksum")

	// add bundle to report
	report.TelemetryBundles = append(
		report.TelemetryBundles,
		*bundle,
	)
	t.Require().NoError(report.UpdateChecksum(), "should be able to update report checksum")

	t.Require().NoError(report.Validate(), "report should be valid")

	tests := []struct {
		name       string
		body       string
		shouldFail bool
	}{
		{
			"dummy test",
			func() string { return "" }(),
			true,
		},

		{
			"Validation with empty reportId",
			func() string {
				copy := copyReport(report)
				copy.Header.ReportId = ""
				content, _ := json.Marshal(&copy)
				return string(content)
			}(),
			true,
		},
		{
			"Validation with empty reportClientId",
			func() string {
				copy := copyReport(report)
				copy.Header.ReportClientId = ""
				content, _ := json.Marshal(&copy)
				return string(content)
			}(),
			true,
		},
		{
			"Validation with empty no reportAnnotations",
			func() string {
				copy := copyReport(report)
				copy.Header.ReportAnnotations = []string{}
				content, _ := json.Marshal(&copy)
				return string(content)
			}(),
			false,
		},
		{
			"Validation with no telemetryBundles",
			func() string {
				copy := copyReport(report)
				copy.TelemetryBundles = []telemetrylib.TelemetryBundle{}
				t.Require().NoError(copy.UpdateChecksum(), "should be able to update report checksum")
				content, _ := json.Marshal(&copy)
				return string(content)
			}(),
			true,
		},
		{
			"Validation with empty report checksum",
			func() string {
				copy := copyReport(report)
				copy.TelemetryBundles = []telemetrylib.TelemetryBundle{}
				copy.Footer.Checksum = ""
				content, _ := json.Marshal(&copy)
				return string(content)
			}(),
			true,
		},
		{
			"Validation with empty bundleId",
			func() string {
				copy := copyReport(report)
				copy.TelemetryBundles[0].Header.BundleId = ""
				t.Require().NoError(copy.TelemetryBundles[0].UpdateChecksum(), "should be able to update bundle checksum")
				t.Require().NoError(copy.UpdateChecksum(), "should be able to update report checksum")
				content, _ := json.Marshal(&copy)
				return string(content)
			}(),
			true,
		},
		{
			"Validation with empty bundleClientId",
			func() string {
				copy := copyReport(report)
				copy.TelemetryBundles[0].Header.BundleClientId = ""
				t.Require().NoError(copy.TelemetryBundles[0].UpdateChecksum(), "should be able to update bundle checksum")
				t.Require().NoError(copy.UpdateChecksum(), "should be able to update report checksum")
				content, _ := json.Marshal(&copy)
				return string(content)
			}(),
			true,
		},
		{
			"Validation with empty bundleAnnotations",
			func() string {
				copy := copyReport(report)
				copy.TelemetryBundles[0].Header.BundleAnnotations = []string{}
				t.Require().NoError(copy.TelemetryBundles[0].UpdateChecksum(), "should be able to update bundle checksum")
				t.Require().NoError(copy.UpdateChecksum(), "should be able to update report checksum")
				content, _ := json.Marshal(&copy)
				return string(content)
			}(),
			false,
		},
		{
			"Validation with empty telemetryDataItems",
			func() string {
				copy := copyReport(report)
				copy.TelemetryBundles[0].TelemetryDataItems = []telemetrylib.TelemetryDataItem{}
				t.Require().NoError(copy.TelemetryBundles[0].UpdateChecksum(), "should be able to update bundle checksum")
				t.Require().NoError(copy.UpdateChecksum(), "should be able to update report checksum")
				content, _ := json.Marshal(&copy)
				return string(content)
			}(),
			true,
		},
		{
			"Validation with empty bundle checksum",
			func() string {
				copy := copyReport(report)
				copy.TelemetryBundles[0].TelemetryDataItems = []telemetrylib.TelemetryDataItem{}
				copy.TelemetryBundles[0].Footer.Checksum = ""
				t.Require().NoError(copy.UpdateChecksum(), "should be able to update report checksum")
				content, _ := json.Marshal(&copy)
				return string(content)
			}(),
			true,
		},
		{
			"Validation with empty telemetryId",
			func() string {
				copy := copyReport(report)
				copy.TelemetryBundles[0].TelemetryDataItems[0].Header.TelemetryId = ""
				t.Require().NoError(copy.TelemetryBundles[0].TelemetryDataItems[0].UpdateChecksum(), "should be able to update data item checksum")
				t.Require().NoError(copy.TelemetryBundles[0].UpdateChecksum(), "should be able to update bundle checksum")
				t.Require().NoError(copy.UpdateChecksum(), "should be able to update report checksum")
				content, _ := json.Marshal(&copy)
				return string(content)
			}(),
			true,
		},
		{
			"Validation with empty telemetryAnnotations",
			func() string {
				copy := copyReport(report)
				copy.TelemetryBundles[0].TelemetryDataItems[0].Header.TelemetryAnnotations = []string{}
				t.Require().NoError(copy.TelemetryBundles[0].TelemetryDataItems[0].UpdateChecksum(), "should be able to update data item checksum")
				t.Require().NoError(copy.TelemetryBundles[0].UpdateChecksum(), "should be able to update bundle checksum")
				t.Require().NoError(copy.UpdateChecksum(), "should be able to update report checksum")
				content, _ := json.Marshal(&copy)
				return string(content)
			}(),
			false,
		},
		{
			"Validation with empty JSON object for telemetryData",
			func() string {
				copy := copyReport(report)
				copy.TelemetryBundles[0].TelemetryDataItems[0].TelemetryData = []byte("{}")
				t.Require().NoError(copy.TelemetryBundles[0].TelemetryDataItems[0].UpdateChecksum(), "should be able to update data item checksum")
				t.Require().NoError(copy.TelemetryBundles[0].UpdateChecksum(), "should be able to update bundle checksum")
				t.Require().NoError(copy.UpdateChecksum(), "should be able to update report checksum")
				content, _ := json.Marshal(&copy)
				return string(content)
			}(),
			false,
		},
		{
			"Validation with empty data item checksum",
			func() string {
				copy := copyReport(report)
				copy.TelemetryBundles[0].TelemetryDataItems[0].Footer.Checksum = ""
				t.Require().NoError(copy.TelemetryBundles[0].UpdateChecksum(), "should be able to update bundle checksum")
				t.Require().NoError(copy.UpdateChecksum(), "should be able to update report checksum")
				content, _ := json.Marshal(&copy)
				return string(content)
			}(),
			false,
		},
	}

	for _, tt := range tests {
		t.Run("Report Telemetry "+tt.name, func() {
			// Test the wrapper.reportTelemetry handler
			// Create a POST request with the necessary body

			rr, err := postToReportTelemetryHandler(tt.body, "", true, t)
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

func createReportPayload(customer_id string) (reportPayload string, err error) {
	// Create 2 dataitems
	telemetryType := types.TelemetryType("SLE-SERVER-Test")
	itags1 := types.Tags{types.Tag("ikey1=ivalue1"), types.Tag("ikey2")}
	itags2 := types.Tags{types.Tag("ikey1=ivalue1")}
	payload := types.NewTelemetryBlob([]byte(`{
		"ItemA": 1,
		"ItemB": "b",
		"ItemC": "c"
	}`))

	item1, err := telemetrylib.NewTelemetryDataItem(telemetryType, itags1, payload)
	if err != nil {
		return "", err
	}

	item2, err := telemetrylib.NewTelemetryDataItem(telemetryType, itags2, payload)
	if err != nil {
		return "", err
	}

	client_id := uuid.New().String()

	// Create 1 bundle
	btags1 := types.Tags{types.Tag("bkey1=bvalue1"), types.Tag("bkey2")}
	bundle1, err := telemetrylib.NewTelemetryBundle(client_id, customer_id, btags1)
	if err != nil {
		return "", err
	}

	// add the two items to the bundle
	bundle1.TelemetryDataItems = append(bundle1.TelemetryDataItems, *item1)
	bundle1.TelemetryDataItems = append(bundle1.TelemetryDataItems, *item2)

	// update the checksum
	err = bundle1.UpdateChecksum()
	if err != nil {
		return "", err
	}

	// Create 1 report
	rtags1 := types.Tags{types.Tag("rkey1=rvalue1"), types.Tag("rkey2")}
	report1, err := telemetrylib.NewTelemetryReport(client_id, rtags1)
	if err != nil {
		return "", err
	}

	report1.TelemetryBundles = append(report1.TelemetryBundles, *bundle1)

	// update the checksum
	err = report1.UpdateChecksum()
	if err != nil {
		return "", err
	}

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
