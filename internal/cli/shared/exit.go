// Package shared defines the CLI exit contract.
package shared

import "errors"

type ExitError struct{ Code int }

func (e ExitError) Error() string { return "collection has incomplete sources" }

// Code maps fatal errors to 1 and persisted partial collection to 2.
func Code(err error) int {
	if err == nil {
		return 0
	}
	var exit ExitError
	if errors.As(err, &exit) {
		return exit.Code
	}
	return 1
}
