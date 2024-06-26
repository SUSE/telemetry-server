package app

import (
	"database/sql"
	"fmt"
	"log/slog"
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

func (s *TelemetryRowXformMap) isSetup(caller string) (err error) {
	if len(s.handlers) < 1 {
		err = fmt.Errorf("no xform handlers registered")
	} else if _, ok := s.handlers[DEF_HANDLER]; !ok {
		err = fmt.Errorf("no default xform handler registered")
	}

	if err != nil {
		slog.Error(err.Error(), slog.String("caller", caller))
	}
	return
}

func (s *TelemetryRowXformMap) SetupDB(db *sql.DB) (err error) {
	if err := s.isSetup("SetupDB"); err != nil {
		return err
	}

	for ttype, handler := range s.handlers {
		if err := handler.SetupDB(db); err != nil {
			slog.Error("xform handler.SetupDB() failed", slog.String("telemetryType", ttype), slog.String("error", err.Error()))
			return err
		}
	}

	return
}

func (s *TelemetryRowXformMap) Get(telemetryType string) (handler TelemetryDataRow) {
	if err := s.isSetup("SetupDB"); err != nil {
		// something is very wrong
		panic(err)
	}

	handler, ok := s.handlers[telemetryType]
	if !ok {
		// we already validated that a DEF_HANDLER entry exists above
		handler = s.handlers[DEF_HANDLER]
	}
	return
}

func (s *TelemetryRowXformMap) Register(telemetryType string, handler TelemetryDataRow) {
	// allocate s.handlers if needed
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
