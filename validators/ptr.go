// Package validators holds the input checks and small conversions shared by
// route handlers.
//
// It covers the checks that are policy rather than syntax — blacklisted
// words, permissible extra links, staff permission lookups — alongside the
// pointer and UUID helpers that request payloads need. Anything here may be
// called before a request is trusted, so nothing in this package assumes its
// input is well-formed.
package validators

var TruePtr = Pointer(true)
var FalsePtr = Pointer(false)

func Pointer[T any](v T) *T {
	return &v
}
