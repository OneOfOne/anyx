package anyx

import (
	"fmt"
	"reflect"
)

// ValPtr returns a pointer to the given value
func ValPtr[T any](v T) *T {
	return &v
}

func SetIfNil[T any](dst **T, val T) (ok bool) {
	if *dst == nil {
		*dst = &val
	}
	return
}

func ConvertSlice(src any, dstStruct any) any {
	rv, dv := indirectValue(reflect.ValueOf(src)), reflect.TypeOf(dstStruct)
	if rv.Kind() != reflect.Slice || indirectValue(rv.Elem()).Kind() != reflect.Struct {
		panic("src must be a slice of a structs")
	}
	switch dv.Kind() {
	case reflect.Ptr:
		if dv.Elem().Kind() != reflect.Struct {
			panic("dstStruct must be a struct or a pointer to a struct")
		}
	case reflect.Struct:
	default:
		panic("dstStruct must be a struct or a pointer to a struct")
	}
	// TODO actual work?
	return src
}

func GroupBy(src any, name string, skipZeroValue bool) (any, error) {
	var (
		v     = indirectValue(reflect.ValueOf(src))
		isPtr bool
		k     reflect.Kind
		et    reflect.Type
		sf    reflect.StructField
		out   reflect.Value
		mkey  reflect.Value
	)
	switch k = v.Kind(); k {
	case reflect.Slice, reflect.Map, reflect.Array:
		switch et = indirectType(v.Type().Elem()); et.Kind() {
		case reflect.Struct:
			sf, _ = et.FieldByName(name)
			if sf.Index == nil {
				return nil, fmt.Errorf("%v doesn't have field %s", et, name)
			}

			isPtr = sf.Type.Kind() == reflect.Ptr
			out = reflect.MakeSlice(reflect.SliceOf(sf.Type), 0, v.Len())
		case reflect.Map:
			mkey = reflect.ValueOf(name)
			et := et.Elem()
			isPtr = et.Kind() == reflect.Ptr
			out = reflect.MakeSlice(reflect.SliceOf(et), 0, v.Len())
		}
	}

	if !out.IsValid() {
		return nil, fmt.Errorf("unexpected type %T", src)
	}

	switch k {
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			rv := indirectValue(v.Index(i))
			if !rv.IsValid() || (skipZeroValue && rv.IsZero()) || (isPtr && rv.IsNil()) {
				continue
			}

			if sf.Index != nil {
				rv = rv.FieldByIndex(sf.Index)
			} else {
				rv = rv.MapIndex(mkey)
			}
			out = reflect.Append(out, rv)
		}
	case reflect.Map:
		for it := v.MapRange(); it.Next(); {
			rv := indirectValue(it.Value())
			if !rv.IsValid() || (skipZeroValue && rv.IsZero()) || (isPtr && rv.IsNil()) {
				continue
			}
			if sf.Index != nil {
				rv = rv.FieldByIndex(sf.Index)
			} else {
				rv = rv.MapIndex(mkey)
			}
			out = reflect.Append(out, rv)
		}
	default:
		panic("not reachable")
	}
	return out.Interface(), nil
}

func indirectType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func indirectValue(t reflect.Value) reflect.Value {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func structIndex(t reflect.Type) map[any][]int {
	t = indirectType(t)

	if t.Kind() != reflect.Struct {
		return nil
	}

	structFields.RLock()
	m := structFields.m[t]
	structFields.RUnlock()
	if m != nil {
		return m
	}

	structFields.Lock()
	defer structFields.Unlock()
	if m = structFields.m[t]; m != nil {
		return m
	}
	m = make(map[any][]int, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		fld := t.Field(i)
		if fld.PkgPath != "" {
			continue
		}
		n := fld.Name
		if n == "" && fld.Anonymous {
			ft := indirectType(fld.Type)
			n = ft.Name()
		}
		m[n] = fld.Index
	}
	structFields.m[t] = m
	return m
}
