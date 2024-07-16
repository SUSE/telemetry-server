package app

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthManager struct {
	config       *AuthConfig
	secret       []byte
	duration     time.Duration
	issuer       string
	methods      []jwt.SigningMethod
	validMethods []string
}

var durationSfxMap = map[string]time.Duration{
	"s": time.Second,
	"m": time.Minute,
	"h": time.Hour,
	"d": time.Hour * 24,
	"w": time.Hour * 24 * 7,
}

func validDurationSuffixes() (sfxs string) {
	for k := range durationSfxMap {
		sfxs += k
	}
	return
}

// use 1 week as default time duration
const DEFAULT_AUTH_TIME_DURATION time.Duration = time.Hour * 24 * 7

// authDuration is a helper function that converts an auth di
func authDuration(cfgDuration string) (timeDuration time.Duration, err error) {
	// support s for seconds, m for minutes, h for hours, d for days and
	// w for weeks, with no suffix meaning seconds

	// strip off surrounding whitespace
	stripped := strings.TrimSpace(cfgDuration)

	// use the default duration if none specified
	if stripped == "" {
		stripped = DEF_AUTH_DURATION
	}

	sfx := strings.TrimLeft(stripped, " +-0123456789")
	digits := strings.TrimSpace(strings.TrimSuffix(stripped, sfx))

	slog.Debug("timeDuration", slog.String("cfgDuration", cfgDuration), slog.String("stripped", stripped), slog.String("digits", digits), slog.String("sfx", sfx))

	// convert the digits to an int64
	durationCount, err := strconv.ParseInt(digits, 0, 64)
	if err != nil {
		return
	}

	// duration must be > 0
	if durationCount <= 0 {
		err = fmt.Errorf(
			"invalid auth.duration value '%s', must be greater than 0",
			cfgDuration,
		)
		return
	}

	// assume seconds if no suffix specified
	if sfx == "" {
		sfx = "s"
	}

	// determine the suffix multiplier
	sfxMult, ok := durationSfxMap[sfx]
	if !ok {
		err = fmt.Errorf(
			"invalid auth.duration suffix '%s', must be one of [%s]",
			sfx,
			validDurationSuffixes(),
		)
		return
	}

	timeDuration = time.Duration(durationCount) * sfxMult

	return
}

func NewAuthManager(ac *AuthConfig) (am *AuthManager, err error) {
	am = new(AuthManager)
	am.secret, err = base64.StdEncoding.DecodeString(ac.Secret)
	if err != nil {
		slog.Error("config auth.secret must be a valid base64 encoded value")
		return nil, err
	}

	am.duration, err = authDuration(ac.Duration)
	if err != nil {
		slog.Error(
			"config auth.duration invalid",
			slog.String("auth.duration", ac.Duration),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	if ac.Issuer != "" {
		am.issuer = ac.Issuer
	} else {
		am.issuer = "telemetry-service-gateway"
	}

	am.methods = []jwt.SigningMethod{
		// first method is the method used for newly generated tokens,
		// remaining methods are valid for existing tokens. Add new
		// preferred methods to start of list
		jwt.SigningMethodHS512,
		jwt.SigningMethodHS384,
		jwt.SigningMethodHS256,
	}

	// generate list of valid methods
	for _, m := range am.methods {
		am.validMethods = append(am.validMethods, m.Alg())
	}

	am.config = ac

	return
}

func (am *AuthManager) newExpirationFrom(t time.Time) (exp *jwt.NumericDate) {
	return jwt.NewNumericDate(t.Add(am.duration))
}

func (am *AuthManager) newExpiration() (exp *jwt.NumericDate) {
	return am.newExpirationFrom(time.Now())
}

func (am *AuthManager) Issuer() string {
	return am.issuer
}

func (am *AuthManager) SigningMethod() jwt.SigningMethod {
	return am.methods[0]
}

func (am *AuthManager) ValidMethods() []string {
	return am.validMethods
}

func (am *AuthManager) Subject(sub any) string {
	return fmt.Sprintf("%v", sub)
}

func (am *AuthManager) CreateToken() (tokenString string, err error) {
	token := jwt.NewWithClaims(
		am.SigningMethod(),
		jwt.MapClaims{
			"exp": am.newExpiration(),
			"iss": am.Issuer(),
		},
	)

	tokenString, err = token.SignedString(am.secret)
	if err != nil {
		slog.Error("jwt token signing failed", slog.String("error", err.Error()))
	}

	return
}

func (am *AuthManager) VerifyToken(tokenString string) (err error) {
	_, err = jwt.Parse(
		tokenString,
		func(*jwt.Token) (any, error) { return am.secret, nil },
		jwt.WithValidMethods(am.ValidMethods()),
		jwt.WithIssuer(am.Issuer()),
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		slog.Error("token parse failed", slog.String("tokenString", tokenString))
		return
	}

	return
}
