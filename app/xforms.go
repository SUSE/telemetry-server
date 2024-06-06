package app

import (
	"database/sql"
	"log"
)

const (
	DEF_HANDLER string = "__default_row_handler"
)

type TelemetryRowXformMapper interface {
	// Register a default telemetry row handler, used when no type
	// specific handler is available
	SetDefault(handler TelemetryDataRow)

	// Register a row handler for a specific telemetry type
	Register(telemetryType string, handler TelemetryDataRow)

	// Setup DB integration for the registered telemetry row handlers
	SetupDB(db *sql.DB) error

	// Retrieve row handler to use for the specified telemetry type
	Get(telemetryType string) (handler TelemetryDataRow)
}

type TelemetryRowXformMap struct {
	handlers map[string]TelemetryDataRow
}

func (s *TelemetryRowXformMap) SetupDB(db *sql.DB) (err error) {
	if _, ok := s.handlers[DEF_HANDLER]; !ok {
		log.Fatalf("ERR: TelemetryRowXformMap.Get() called default registered")
	}

	for ttype, handler := range s.handlers {
		if err := handler.SetupDB(db); err != nil {
			log.Printf("ERR: SetupDB() failed for handler %q: %s", ttype, err.Error())
			return err
		}
	}

	return
}

func (s *TelemetryRowXformMap) Get(telemetryType string) (handler TelemetryDataRow) {
	defHandler, ok := s.handlers[DEF_HANDLER]
	if !ok {
		log.Fatalf("ERR: TelemetryRowXformMap.Get() called default registered")
	}

	handler, ok = s.handlers[telemetryType]
	if !ok {
		handler = defHandler
	}
	return
}

func (s *TelemetryRowXformMap) Register(telemetryType string, handler TelemetryDataRow) {
	if s.handlers == nil {
		s.handlers = make(map[string]TelemetryDataRow)
	}
	s.handlers[telemetryType] = handler
}

func (s *TelemetryRowXformMap) SetDefault(handler TelemetryDataRow) {
	s.Register(DEF_HANDLER, handler)
}

// validate that TelemetryRowXformMap provides TelemetryRowXformMapper
var _ TelemetryRowXformMapper = (*TelemetryRowXformMap)(nil)
