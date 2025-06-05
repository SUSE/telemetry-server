package dialect

import (
	"fmt"
)

// SQL Placeholder support
// SQLite uses `?` for a placeholder, PostgreSQL uses `$1`, `$2`, ...
type Placeholder chan string

// retrieve the next placeholder value, panic if channel closed
func (p Placeholder) Next() string {
	next, ok := <-p
	if !ok {
		panic(fmt.Errorf("placeholder next call after channel was closed"))
	}

	return next
}

type phPrinter func(int) string

// a placeholder generator type
type PlaceholderGenerator func(int) Placeholder

func phGenerator(printer phPrinter, numFields int) Placeholder {
	ph := make(chan string)

	// create a go routine to generate placeholder values
	go func() {
		counter := 1
		for {
			// close the channel if all requested values have been generated
			if numFields == 0 {
				close(ph)
				return
			}
			ph <- printer(counter)
			counter++
			numFields--
		}
	}()
	return ph
}
