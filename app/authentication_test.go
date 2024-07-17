package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AuthenticationTestSuite struct {
	suite.Suite
}

func (t *AuthenticationTestSuite) TestTimeDurations() {
	tests := []struct {
		name        string
		config      string
		expectValue time.Duration
		expectFail  bool
	}{
		{
			name:        "Empty duration should use default",
			config:      "",
			expectValue: time.Hour * 24 * 7,
		},
		{
			name:        "Blank duration should use default",
			config:      " ",
			expectValue: time.Hour * 24 * 7,
		},
		{
			name:        "Valid number, no suffix",
			config:      "22",
			expectValue: time.Second * 22,
		},
		{
			name:        "Valid number of seconds",
			config:      "31s",
			expectValue: time.Second * 31,
		},
		{
			name:        "Valid number of minutes",
			config:      "6m",
			expectValue: time.Minute * 6,
		},
		{
			name:        "Valid number of hours",
			config:      "99h",
			expectValue: time.Hour * 99,
		},
		{
			name:        "Valid number of days",
			config:      "222d",
			expectValue: time.Hour * 24 * 222,
		},
		{
			name:        "Valid number of weeks",
			config:      "9w",
			expectValue: time.Hour * 24 * 7 * 9,
		},
		{
			name:       "Invalid number",
			config:     "0x1a",
			expectFail: true,
		},
		{
			name:       "Invalid suffix",
			config:     "1ms",
			expectFail: true,
		},
		{
			name:       "Suffix only, no number",
			config:     "d",
			expectFail: true,
		},
		{
			name:       "Invalid duration, zero",
			config:     "0",
			expectFail: true,
		},
		{
			name:       "Invalid duration, negative",
			config:     "-1",
			expectFail: true,
		},
	}

	for _, tt := range tests {
		t.Run("Validating config auth duration parsing"+tt.name, func() {

			duration, err := authDuration(tt.config)
			if tt.expectFail {
				assert.Error(t.T(), err, "authDuration(%s) should have failed", tt.config)
			} else {
				assert.Equal(t.T(), tt.expectValue, duration, "time.Duration(%s) returned wrong value", tt.config)
			}
		})
	}
}

func TestAuthenticationTestSuite(t *testing.T) {
	suite.Run(t, new(AuthenticationTestSuite))
}
