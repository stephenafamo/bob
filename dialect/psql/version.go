package psql

import "context"

// VersionKey is a context key for storing the database version.
// Use SetVersion to set the version in context.
type VersionKey struct{}

// SetVersion sets the major version (e.g., 15, 16, 17) in the context.
// This is used to enable version-specific features like MERGE with RETURNING (version 17+).
//
// Example:
//
//	ctx := psql.SetVersion(ctx, 17)
func SetVersion(ctx context.Context, version int) context.Context {
	return context.WithValue(ctx, VersionKey{}, version)
}

// GetVersion returns the major version from the context.
// Returns 0 if the version is not set.
func GetVersion(ctx context.Context) int {
	if v, ok := ctx.Value(VersionKey{}).(int); ok {
		return v
	}
	return 0
}

// VersionAtLeast checks if the version in context is at least the given version.
// Returns false if version is not set in context.
func VersionAtLeast(ctx context.Context, minVersion int) bool {
	return GetVersion(ctx) >= minVersion
}
