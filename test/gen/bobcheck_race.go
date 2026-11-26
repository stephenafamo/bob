//go:build race

package testgen

// SkipBobcheck avoids the go/types race when the test fixture loads generated packages.
const SkipBobcheck = true
