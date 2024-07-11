package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type DbPlaceholderTestSuite struct {
	suite.Suite
}

func (t *DbPlaceholderTestSuite) TestDbPlaceholderPrinters() {
	tests := []struct {
		name     string
		printer  phPrinter
		field    int
		expected string
	}{
		{
			name:     "dollarCount",
			printer:  dollarCount,
			field:    99,
			expected: "$99",
		},
		{
			name:     "questionMark",
			printer:  questionMark,
			field:    1,
			expected: "?",
		},
	}
	for _, tt := range tests {
		t.Run("Validating placeholder printer "+tt.name, func() {
			assert.Equal(t.T(), tt.expected, tt.printer(tt.field), "placeholder printer returned wrong value")
		})
	}
}

func (t *DbPlaceholderTestSuite) TestDbPlaceolders() {
	tests := []struct {
		name      string
		printer   phPrinter
		generator PlaceholderGenerator
		count     int
	}{
		{
			name:      "DollarCounter",
			printer:   dollarCount,
			generator: DollarCounter,
			count:     20,
		},
		{
			name:      "QuestionMarker",
			printer:   questionMark,
			generator: QuestionMarker,
			count:     5,
		},
	}

	for _, tt := range tests {
		t.Run("Validating placeholder generator "+tt.name, func() {
			ph := tt.generator(tt.count)
			for i := 1; i <= tt.count; i++ {
				assert.Equal(t.T(), tt.printer(i), ph.Next(), "ph.Next() returned wrong value")
			}
			assert.Panics(t.T(), func() { _ = ph.Next() }, "ph.Next() should panic if all placeholders have been generated")
		})
	}
}

func TestDbPlaceholdersTestSuite(t *testing.T) {
	suite.Run(t, new(DbPlaceholderTestSuite))
}
