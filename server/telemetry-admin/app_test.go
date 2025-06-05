package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/SUSE/telemetry-server/app"
	"github.com/SUSE/telemetry-server/app/config"
	"github.com/SUSE/telemetry/pkg/types"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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
	s.path, err = os.MkdirTemp("", "telemetry-admin-test-")
	require.NoError(s.T(), err)
	tmpfile, err := os.CreateTemp(s.path, "*.yaml")
	require.NoError(s.T(), err)

	content := `
---
api:
  host: localhost
  port: 9998
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
		Value:  "56dad39883e6b69e68523e8991a9237422a13031fd5f136286045a3e9b79f3ce",
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

// Verify correct handling of /healthz request
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

func TestAppTestSuite(t *testing.T) {
	suite.Run(t, new(AppTestSuite))
}
