package dialect

// print question marks for every placeholder
func questionMark(counter int) string {
	return "?"
}

// verify that questionMark conforms to phPrinter type
var _ phPrinter = questionMark

// create a placeholder generator that returns question marks
func QuestionMarker(numFields int) Placeholder {
	return phGenerator(questionMark, numFields)
}

// verify that QuestionMarker conforms to PlaceholderGenerator type
var _ PlaceholderGenerator = QuestionMarker
