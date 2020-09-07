package anyx

import (
	"reflect"

	"golang.org/x/xerrors"
)

func GroupBy(src interface{}, name string, skipZeroValue bool) (interface{}, error) {
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
				return nil, xerrors.Errorf("%v doesn't have field %s", et, name)
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
		return nil, xerrors.Errorf("unexpected type %T", src)
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
