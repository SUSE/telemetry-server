package app

import (
	"fmt"

	"github.com/SUSE/telemetry-server/app/config"
	_ "github.com/mattn/go-sqlite3"
)

// ServerAddress is a struct tracking the server address
type ServerAddress struct {
	Hostname string
	Port     int
}

func (s ServerAddress) String() string {
	return fmt.Sprintf("%s:%d", s.Hostname, s.Port)
}

func (s *ServerAddress) Setup(api config.APIConfig) {
	s.Hostname, s.Port = api.Host, api.Port
}
