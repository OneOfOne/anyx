package anyx // import "go.oneofone.dev/anyx"

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"sync"
	"time"
)

var (
	DefaultTimeLayouts      = [...]string{time.RFC3339Nano, time.RFC1123, time.RFC1123Z}
	DefaultShortTimeLayouts = [...]string{"2006-01-02", "2006/01/02"}

	structFields struct {
		sync.RWMutex
		m map[reflect.Type]map[any][]int
	}
)

func ValueOf(v any) (a Value) {
	a.Set(v)
	return
}

// Map returns Any map[string]Any using pairs of (key.(string), val).
func Map(pairs ...any) (a Value) {
	if len(pairs) == 0 {
		return Value{}
	}
	if len(pairs)%2 != 0 {
		panic("len(pairs) % 2 != 0")
	}
	v := make(map[string]Value, len(pairs)/2)
	for i := 0; i < len(pairs)-1; i += 2 {
		v[pairs[i].(string)] = ValueOf(pairs[i+1])
	}
	a.v = v
	return a
}

func Slice(vals ...any) (a Value) {
	v := make([]Value, 0, len(vals))
	for _, a := range vals {
		v = append(v, ValueOf(a))
	}
	a.v = v
	return
}

type Value struct {
	v any
}

// Len returns the length of the underlying map/slice.
// if a isn't a map or a slice, Len returns -1.
func (a Value) Len() int {
	switch v := a.v.(type) {
	case interface{ Len() int }:
		return v.Len()
	case []Value:
		return len(v)
	case map[string]Value:
		return len(v)
	default:
		var vv reflect.Value
		if v, ok := a.v.(reflect.Value); ok {
			vv = v
		} else {
			vv = reflect.ValueOf(a.v)
		}
		vv = reflect.Indirect(vv)
		switch vv.Kind() {
		case reflect.Map, reflect.Slice, reflect.Array:
			return vv.Len()
		}
	}

	return 0
}

// ForEach will loop through slices, arrays (key will be the index, int), maps or structs
// Note that on a struct, it will skip zero-valued fields (0, "", nil)
func (a Value) ForEach(fn func(key any, value Value) (ok bool)) {
	switch v := a.v.(type) {
	case interface{ ForEach(func(any, Value) bool) }:
		v.ForEach(fn)
	case []Value:
		for i := range v {
			if fn(i, v[i]) {
				return
			}
		}
	case map[string]Value:
		for k := range v {
			if fn(k, v[k]) {
				return
			}
		}
	case reflect.Value:
		switch v = reflect.Indirect(v); v.Kind() {
		case reflect.Slice, reflect.Array:
			for i := 0; i < v.Len(); i++ {
				if fn(i, Value{v: v.Index(i)}) {
					return
				}
			}

		case reflect.Map:
			for it := v.MapRange(); it.Next(); {
				if fn(it.Key().Interface(), ValueOf(it.Value())) {
					return
				}
			}
		case reflect.Struct:
			t := v.Type()
			for i := 0; i < t.NumField(); i++ {
				kf := t.Field(i)
				n := kf.Name
				if kf.Anonymous {
					n = kf.Type.Name()
				}
				v := v.Field(i)
				if v.IsZero() {
					continue
				}
				if fn(n, ValueOf(v)) {
					return
				}
			}
		}
	}
}

// Get will nest for all the given keys, for example:
// Map("key", Slice(1, Map("key", Slice(42)), 3)).Get("key", 1, "key", 0).Int() === 42
func (a Value) Get(key any) (_ Value) {
	switch v := a.v.(type) {
	case interface{ Get(any) any }:
		return Value{v: v.Get(key)}
	case []Value:
		a = v[key.(int)]
	case map[string]Value:
		a = v[key.(string)]
	default:
		rv := indexReflect(a.v, key)
		if rv == nil {
			return
		}
		a = ValueOf(rv)
	}

	return a
}

func (a Value) Has(key any) bool {
	switch v := a.v.(type) {
	case interface{ Has(any) bool }:
		return v.Has(key)
	case map[string]Value:
		k, _ := key.(string)
		_, ok := v[k]
		return ok
	case []Value:
		i, _ := key.(int)
		return i >= 0 && i < len(v)
	case reflect.Value:
		v = reflect.Indirect(v)
		switch v.Kind() {
		case reflect.Map:
			return v.MapIndex(reflect.ValueOf(key)).IsValid()
		case reflect.Slice, reflect.Array:
			n := ValueOf(key).Int()
			return n >= 0 && n < int64(v.Len())
		case reflect.Struct:
			return structIndex(v.Type())[key] != nil
		}
	}

	return false
}

func (a Value) Keys() (out []Value) {
	switch v := a.v.(type) {
	case interface{ Keys() []Value }:
		return v.Keys()
	case map[string]Value:
		out = make([]Value, 0, len(v))
		for k := range v {
			out = append(out, Value{k})
		}
	case reflect.Value:
		v = reflect.Indirect(v)
		switch v.Kind() {
		case reflect.Map:
			mk := v.MapKeys()
			out = make([]Value, 0, len(mk))
			for _, k := range mk {
				out = append(out, Value{k.Interface()})
			}
		case reflect.Struct:
			t := v.Type()
			out = make([]Value, 0, t.NumField())
			for i := 0; i < cap(out); i++ {
				if f := t.Field(i); f.Name != "" {
					out = append(out, Value{f.Name})
				}
			}
		}
	}

	return
}

func (a Value) Values() (out []Value) {
	switch v := a.v.(type) {
	case interface{ Values() []Value }:
		return v.Values()
	case map[string]Value:
		out = make([]Value, 0, len(v))
		for _, vv := range v {
			out = append(out, Value{vv})
		}
	case reflect.Value:
		v = reflect.Indirect(v)
		switch v.Kind() {
		case reflect.Map:
			mk := v.MapKeys()
			out = make([]Value, 0, len(mk))
			for _, k := range mk {
				out = append(out, Value{v.MapIndex(k).Interface()})
			}
		case reflect.Struct:
			t := v.Type()
			out = make([]Value, 0, t.NumField())
			for i := 0; i < cap(out); i++ {
				if f := t.Field(i); f.Name != "" {
					out = append(out, Value{f.Name})
				}
			}
		}
	}

	return
}

// String returns the string of the underlying Any
// if always is true, it'll call fmt.Sprint
func (a Value) String(always bool) string {
	switch v := a.v.(type) {
	case nil:
		return "nil"
	case string:
		return v
	case json.RawMessage:
		return trim(v)
	case reflect.Value:
		return fmt.Sprintf("%v", v.Interface())
	case fmt.Stringer:
		return v.String()
	case error:
		return v.Error()
	}

	if always {
		return fmt.Sprint(a.v)
	}
	return ""
}

func (a Value) Int() int64 {
	var s string
	switch v := a.v.(type) {
	case json.RawMessage:
		s = trim(v)
	case json.Number:
		s = string(v)
	case int64:
		return v
	case uint64:
		return int64(v)
	case float64:
		return int64(v)
	case reflect.Value:
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return v.Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return int64(v.Uint())
		case reflect.Float32, reflect.Float64:
			return int64(v.Float())
		case reflect.Bool:
			if v.Bool() {
				return 1
			}
			return 0
		case reflect.String:
			s = v.String()
		}
	}

	if s == "" {
		return 0
	}
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func (a Value) Uint() uint64 {
	var s string
	switch v := a.v.(type) {
	case json.RawMessage:
		s = trim(v)
	case json.Number:
		s = string(v)
	case uint64:
		return v
	case int64:
		return uint64(v)
	case float64:
		return uint64(v)
	case reflect.Value:
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return uint64(v.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return v.Uint()
		case reflect.Float32, reflect.Float64:
			return uint64(v.Float())
		case reflect.Bool:
			if v.Bool() {
				return 1
			}
			return 0
		case reflect.String:
			s = v.String()
		}
	}
	if s == "" {
		return 0
	}
	n, _ := strconv.ParseUint(s, 10, 64)
	return n
}

func (a Value) Float() float64 {
	var s string
	switch v := a.v.(type) {
	case json.RawMessage:
		s = trim(v)
	case json.Number:
		s = string(v)
	case float64:
		return v
	case uint64:
		return float64(v)
	case int64:
		return float64(v)
	case reflect.Value:
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return float64(v.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return float64(v.Uint())
		case reflect.Float32, reflect.Float64:
			return v.Float()
		case reflect.Bool:
			if v.Bool() {
				return 1
			}
			return 0
		case reflect.String:
			s = v.String()
		}
	}
	if s == "" {
		return 0
	}
	n, _ := strconv.ParseFloat(s, 64)
	return n
}

func (a Value) Bool() bool {
	var s string
	switch v := a.v.(type) {
	case json.RawMessage:
		s = trim(v)
	case string:
		s = v
	case bool:
		return v
	case float64:
		return v != 0
	case uint64:
		return v != 0
	case int64:
		return v != 0
	case reflect.Value:
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return v.Int() != 0
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return v.Uint() != 0
		case reflect.Float32, reflect.Float64:
			return v.Float() != 0
		case reflect.Bool:
			return v.Bool()
		case reflect.String:
			s = v.String()
		}
	}
	if s == "" {
		return false
	}
	n, _ := strconv.ParseBool(s)
	return n
}

func (a Value) IsNumber() bool {
	var s string
	switch v := a.v.(type) {
	case int, uint, int32, uint32, int64, uint64, float32, float64:
		return true
	case json.RawMessage:
		s = trim(v)
	case json.Number:
		return true
	case reflect.Value:
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			return true
		}
	}

	if len(s) == 0 {
		return false
	}

	switch s[0] {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-', '+':
		return true
	default:
		return false
	}
}

func (a Value) Time(layouts ...string) (t time.Time) {
	const ms = int64(1e12)
	const ns = int64(1e18)

	var ok bool
	if t, ok = a.v.(time.Time); ok {
		return
	}

	if n := a.Int(); n > 0 {
		if n > ns {
			return time.Unix(0, n)
		}
		if n > ms {
			n /= 1000 // js date
		}
		return time.Unix(n, 0)
	}

	if s := a.String(false); s != "" {
		if len(layouts) == 0 {
			if len(s) == 10 {
				layouts = DefaultShortTimeLayouts[:]
			} else {
				layouts = DefaultTimeLayouts[:]
			}
		}

		for _, l := range layouts {
			if t, _ = time.Parse(l, s); !t.IsZero() {
				return
			}
		}
	}

	return
}

func (a *Value) Append(val any) {
	if a.v == nil {
		a.v = []Value{}
	}

	switch v := a.v.(type) {
	case interface{ Append(any) }:
		v.Append(val)
	case []Value:
		a.v = append(v, ValueOf(val))
	case reflect.Value:
		a.v = reflect.Append(v, reflect.ValueOf(val))
	default:
		a.v = reflect.Append(reflect.ValueOf(a.v), reflect.ValueOf(val))
	}
}

func (a *Value) SetAt(idx int, val any) {
	if a.v == nil {
		a.v = make([]Value, idx+1)
	}

	switch v := a.v.(type) {
	case interface{ SetAt(int, any) }:
		v.SetAt(idx, val)
	case []Value:
		v[idx].Set(val)
	case reflect.Value:
		v.Index(idx).Set(reflect.ValueOf(val))
	default:
		reflect.ValueOf(v).Index(idx).Set(reflect.ValueOf(val))
	}
}

func (a *Value) SetKeyVal(key, val any) {
	if a.v == nil {
		a.v = map[string]Value{}
	}

	switch v := a.v.(type) {
	case interface{ Set(any, any) }:
		v.Set(key, val)
	case map[string]Value:
		v[key.(string)] = ValueOf(val)
	case []Value:
		v[key.(int)] = ValueOf(val)
	case reflect.Value:
		if m := structIndex(v.Type()); m != nil {
			v.FieldByIndex(m[key.(string)]).Set(reflect.ValueOf(val))
		} else {
			v.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(val))
		}
	default:
		log.Panicf("wrong type %T", a.v)
	}
}

func Set[T any](a *Value, v T) {
	*a = ValueOf(v)
}

// Set sets the current Any to the given value.
func (a *Value) Set(v any) {
	switch v := v.(type) {
	case nil:
		a.v = nil
	case Value:
		a.v = v.v
	case *Value:
		a.v = v.v
	case string:
		a.v = v
	case int:
		a.v = int64(v)
	case float64:
		a.v = v
	case bool:
		a.v = v
	case time.Time:
		a.v = v
	case reflect.Value:
		a.set(v)
	default:
		a.set(reflect.ValueOf(v))
	}
}

func (a *Value) set(rv reflect.Value) {
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		a.v = rv.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		a.v = rv.Uint()
	case reflect.Float32, reflect.Float64:
		a.v = rv.Float()
	case reflect.String:
		a.v = rv.String()
	case reflect.Bool:
		a.v = rv.Bool()
	case reflect.Ptr:
		if rv.IsNil() {
			return
		} else if e := indirectValue(rv); e.Kind() == reflect.Struct {
			a.v = e
		} else {
			a.set(e)
		}
	default:
		a.v = rv
	}
}

func (a Value) Type() string {
	switch v := a.v.(type) {
	case json.Number:
		return "number"
	case reflect.Value:
		return v.Kind().String()
	case nil:
		return "nil"
	default:
		return reflect.ValueOf(v).Kind().String()
	}
}

func (a Value) IsNil() bool {
	return a.v == nil
}

func (a Value) Raw() any {
	return a.v
}

func (a *Value) UnmarshalJSON(b []byte) (err error) {
	switch {
	case len(b) == 0:
		return nil
	case len(b) == 2:
		switch string(b) {
		case "[]":
			a.v = []Value{}
			return
		case "{}":
			a.v = map[string]Value{}
		}
	}

	switch b[0] {
	case '"':
		a.v = string(b[1 : len(b)-1])
	case '[':
		var v []Value
		err = json.Unmarshal(b, &v)
		a.v = v
	case '{':
		var v map[string]Value
		err = json.Unmarshal(b, &v)
		a.v = v
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-', '+', '.':
		a.v = json.Number(b)
	case 't', 'T':
		a.v = true
	case 'f', 'F':
		a.v = false
	default:
		var v any
		err = json.Unmarshal(b, &v)
		a.Set(v)
	}

	if err != nil {
		a.v = nil
	}
	return
}

func (a Value) MarshalJSON() ([]byte, error) {
	if rv, ok := a.v.(reflect.Value); ok {
		return json.Marshal(rv.Interface())
	}
	return json.Marshal(a.v)
}

func (a Value) Format(s fmt.State, c rune) {
	if s.Flag('+') {
		flag := "%v"
		if s.Flag('#') {
			flag = "%+v"
		}
		fmt.Fprintf(s, "Any{%s: "+flag+"}", a.Type(), a.v)
	} else {
		fmt.Fprintf(s, "%v", a.v)
	}
}

func trim(b []byte) string {
	if len(b) > 2 && b[0] == '"' && b[len(b)-1] == '"' {
		b = b[1 : len(b)-1]
	}
	return string(b)
}

func indexReflect(v any, keyOrIndex any) any {
	var rv, ov reflect.Value

	if vv, ok := v.(reflect.Value); ok {
		rv = indirectValue(vv)
	} else {
		rv = indirectValue(reflect.ValueOf(v))
	}

	switch rv.Kind() {
	case reflect.Map:
		ov = rv.MapIndex(reflect.ValueOf(keyOrIndex))
	case reflect.Array, reflect.Slice:
		i, _ := keyOrIndex.(int)
		ov = rv.Index(i)
	case reflect.Struct:
		m := structIndex(rv.Type())
		s, _ := keyOrIndex.(string)
		if f := rv.FieldByIndex(m[s]); f.IsValid() {
			ov = f
		}
	}

	if ov.Kind() != reflect.Invalid {
		return ov.Interface()
	}

	return nil
}

func As[T any](v Value) (out T) {
	switch v := v.v.(type) {
	case T:
		return v
	case reflect.Value:
		return v.Interface().(T)
	default:
		rv := reflect.Indirect(reflect.ValueOf(v))
		return rv.Convert(reflect.TypeOf(out)).Interface().(T)
	}
}
