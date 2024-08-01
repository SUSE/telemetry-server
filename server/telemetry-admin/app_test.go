package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/SUSE/telemetry/pkg/types"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/SUSE/telemetry-server/app"
)

type AppTestSuite struct {
	suite.Suite
	app              *app.App
	config           *app.Config
	router           *mux.Router
	path             string
	authToken        string
	clientInstanceId types.ClientInstanceId
	clientInstIdHash types.ClientInstanceIdHash
	clientId         int64
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
	s.clientInstanceId = `PQR0123456789`
	s.clientInstIdHash = types.ClientInstanceIdHash{
		Method: "sha256",
		Value:  "279b3ce1c73f3598ee36cde0a38fa6687aa33f50935a8b10a0a6608d3084d22a",
	}
	row := s.app.OperationalDB.Conn.QueryRow(
		`INSERT INTO clients(`+
			`clientInstanceId, `+
			`registrationDate, `+
			`authToken) `+
			`VALUES(?, ?, ?) `+
			`RETURNING id`,
		string(s.clientInstanceId),
		"2024-07-01T00:00:00.000000000Z",
		s.authToken,
	)
	if err := row.Scan(&s.clientId); err != nil {
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
