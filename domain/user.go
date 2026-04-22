// Package domain defines the core domain models.
package domain

// User represents a user entity with basic identification fields.
type User struct {
	ID        uint64 `json:"id"`
	UUID      string `json:"uuid"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}
