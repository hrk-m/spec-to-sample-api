// Package domain defines the core domain models.
package domain

import "errors"

var (
	// ErrInternalServerError is returned when an unexpected internal error occurs.
	ErrInternalServerError = errors.New("internal server error")
	// ErrNotFound is returned when a requested resource is not found.
	ErrNotFound = errors.New("your requested item is not found")
	// ErrConflict is returned when there is a data conflict.
	ErrConflict = errors.New("your item already exist")
	// ErrBadParamInput is returned when the given parameter is invalid.
	ErrBadParamInput = errors.New("given param is not valid")
)
