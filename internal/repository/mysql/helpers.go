// Package mysql provides MySQL implementations of repository interfaces.
package mysql

import "strings"

// placeholdersAndArgs builds a comma-separated "?,?,?" placeholder string and the matching args slice
// for use with SQL IN clauses. Returns an empty string and nil when ids is empty;
// callers must guard "IN ()" themselves since MySQL rejects empty IN lists.
func placeholdersAndArgs(ids []uint64) (string, []any) {
	if len(ids) == 0 {
		return "", nil
	}

	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	return strings.Repeat("?,", len(ids)-1) + "?", args
}
