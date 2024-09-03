package validators

var TruePtr = Pointer(true)
var FalsePtr = Pointer(false)

func Pointer[T any](v T) *T {
	return &v
}
