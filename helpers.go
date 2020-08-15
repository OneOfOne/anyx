package any

import "reflect"

// ValPtr returns a pointer to the given value
func ValPtr(v A) A {
	rv := reflect.ValueOf(v)
	if v == nil || !rv.IsValid() {
		var nilp *A
		return nilp
	}
	prv := reflect.New(rv.Type())
	prv.Elem().Set(rv)
	return prv.Interface()
}

// SetIfNil checks if dst is nil, if yes it'll set it to newVal,
// will panic any type mismatch
// example:
// var b *bool // default to true if nil
// SetIfNil(&b, true)
// SetIfNil(&b, 42) // panic: type mismatch
// SetIfNil(b, 42) // panic: dst must be a pointer to a pointer
func SetIfNil(dst, newVal A) (ok bool) {
	drv := reflect.ValueOf(dst).Elem()
	if drv.Kind() != reflect.Ptr || drv.Elem().Kind() != reflect.Ptr {
		panic("dst must be a pointer to a pointer")
	}
	if ok = drv.IsNil(); ok {
		drv.Set(reflect.ValueOf(ValPtr(newVal)))
	}
	return
}
