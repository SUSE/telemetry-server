package dialect

import (
	"fmt"
)

// verify that QuestionMarker conforms to PlaceholderGenerator type
var _ PlaceholderGenerator = QuestionMarker

// print '$n' where n is the counter value for a given placeholder
func dollarCount(counter int) string {
	return fmt.Sprintf("$%d", counter)
}

// verify that dollarCount conforms to phPrinter type
var _ phPrinter = dollarCount

// create a placeholder generator that returns incrementing $n values
func DollarCounter(numFields int) Placeholder {
	return phGenerator(dollarCount, numFields)
}

// verify that DollarCounter conforms to PlaceholderGenerator type
var _ PlaceholderGenerator = DollarCounter
