// Package rest provides HTTP handlers for the API.
package rest

import (
	"strconv"

	"github.com/hrk-m/spec-to-dev-workflow/sample-api/domain"
)

const (
	limitDefault = 500
	limitMin     = 1
	limitMax     = 500
)

// parseLimit parses and validates a limit query parameter.
// Returns limitDefault when s is empty, and ErrBadParamInput when the value is out of [limitMin, limitMax].
func parseLimit(s string) (int, error) {
	if s == "" {
		return limitDefault, nil
	}

	l, err := strconv.Atoi(s)
	if err != nil || l < limitMin || l > limitMax {
		return 0, domain.ErrBadParamInput
	}

	return l, nil
}

// parseOffset parses and validates an offset query parameter.
// Returns 0 when s is empty, and ErrBadParamInput when the value is negative.
func parseOffset(s string) (int, error) {
	if s == "" {
		return 0, nil
	}

	o, err := strconv.Atoi(s)
	if err != nil || o < 0 {
		return 0, domain.ErrBadParamInput
	}

	return o, nil
}

// parsePathID parses and validates a path parameter as a positive uint64 ID.
// Returns ErrBadParamInput when the value is not a valid positive integer.
func parsePathID(s string) (uint64, error) {
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil || id < 1 {
		return 0, domain.ErrBadParamInput
	}

	return id, nil
}
