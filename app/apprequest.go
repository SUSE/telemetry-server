package app

import (
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// AppVars is a map var name to value
type AppVars map[string]string

// AppRequest is a struct tracking the resources associated with handling a request
type AppRequest struct {
	W     http.ResponseWriter
	R     *http.Request
	Vars  AppVars
	Log   *slog.Logger
	Quiet bool
}

func (ar *AppRequest) getReader() (io.ReadCloser, error) {
	// Check the Content-Encoding header
	switch ar.R.Header.Get("Content-Encoding") {
	case "gzip":
		return gzip.NewReader(ar.R.Body)
	case "deflate":
		return zlib.NewReader(ar.R.Body)
	default:
		return ar.R.Body, nil
	}

}

func ReqLogger(r *http.Request) *slog.Logger {
	return slog.Default().With(slog.String("method", r.Method), slog.Any("URL", r.URL))
}

func NewAppRequest(w http.ResponseWriter, r *http.Request, v AppVars) *AppRequest {
	return &AppRequest{
		W:    w,
		R:    r,
		Vars: v,
		Log:  ReqLogger(r),
	}
}

func QuietAppRequest(w http.ResponseWriter, r *http.Request, v AppVars) (ar *AppRequest) {
	ar = NewAppRequest(w, r, v)
	ar.Quiet = true
	return
}

func (ar *AppRequest) GetHeader(header string) (value string) {
	value = ar.R.Header.Get(header)
	ar.Log.Debug("Request header", slog.String(header, value))
	return
}

func (ar *AppRequest) GetAuthorization() string {
	return ar.GetHeader("Authorization")
}

func (ar *AppRequest) GetAuthToken() string {
	return strings.TrimPrefix(ar.GetAuthorization(), "Bearer ")
}

func (ar *AppRequest) GetRegistrationId() string {
	return ar.GetHeader("X-Telemetry-Registration-Id")
}

func (ar *AppRequest) SetHeader(header, value string) {
	ar.Log.Debug("Response header", slog.String(header, value))
	ar.W.Header().Set(header, value)
}

func (ar *AppRequest) ContentType(contentType string) {
	ar.SetHeader("Content-Type", contentType)
}

func (ar *AppRequest) ContentTypeJSON() {
	ar.ContentType("application/json")
}

func (ar *AppRequest) SetWwwAuthenticate(challenge, realm, scope string) {
	ar.SetHeader(
		"WWW-Authenticate",
		fmt.Sprintf(`%s realm="%s" scope="%s"`, challenge, realm, scope),
	)
}

func (ar *AppRequest) SetWwwAuthScope(scope string) {
	ar.SetWwwAuthenticate("Bearer", "suse-telemetry-service", scope)
}

func (ar *AppRequest) SetWwwAuthReauth() {
	ar.SetWwwAuthScope("authenticate")
}

func (ar *AppRequest) SetWwwAuthRegister() {
	ar.SetWwwAuthScope("register")
}

func (ar *AppRequest) Status(statusCode int) {
	ar.Log.Debug("Response status", slog.Int("code", statusCode))
	ar.W.WriteHeader(statusCode)
}

func (ar *AppRequest) StatusInternalServerError() {
	ar.Status(http.StatusInternalServerError)
}

func (ar *AppRequest) Write(data []byte) (code int, err error) {
	ar.Log.Debug("Response write", slog.Int("length", len(data)))
	code, err = ar.W.Write(data)
	return
}

func (ar *AppRequest) ErrorResponse(code int, errorMessage string) {
	ar.Log.Debug("Setting error response", slog.Int("code", code), slog.String("error", errorMessage))
	ar.JsonResponse(code, map[string]string{"error": errorMessage})
}

func (ar *AppRequest) JsonResponse(code int, payload any) {
	respContent, err := json.Marshal(payload)
	if err != nil {
		ar.ErrorResponse(http.StatusInternalServerError, err.Error())
		ar.Log.Error("Payload marshal failed", slog.Int("code", code), slog.String("error", err.Error()))
		return
	}

	ar.ContentTypeJSON()
	ar.Status(code)
	writeCode, err := ar.Write(respContent)
	if err != nil {
		ar.Log.Error("Response write failed", slog.Int("code", code), slog.Int("writeCode", writeCode), slog.String("error", err.Error()))
	} else {
		if !ar.Quiet {
			ar.Log.Info("Response", slog.Int("code", code))
		} else {
			ar.Log.Debug("Response", slog.Int("code", code))
		}
	}
}
