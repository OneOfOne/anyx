package any

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

type A = interface{}

var (
	DefaultTimeLayouts      = [...]string{time.RFC3339Nano, time.RFC1123, time.RFC1123Z}
	DefaultShortTimeLayouts = [...]string{"2006-01-02", "2006/01/02"}
)

func Value(v A) (a Any) {
	a.Set(v)
	return
}

func Slice(vals ...A) (a Any) {
	v := make([]Any, 0, len(vals))
	for _, a := range vals {
		v = append(v, Value(a))
	}
	a.v = v
	return
}

// Map returns Any map[string]Any using pairs of (key.(string), val).
func Map(pairs ...A) (a Any) {
	if len(pairs) == 0 {
		return
	}
	if len(pairs)%2 != 0 {
		panic("len(pairs) % 2 != 0")
	}
	v := make(map[string]Any, len(pairs)/2)
	for i := 0; i < len(pairs)-1; i += 2 {
		v[pairs[i].(string)] = Value(pairs[i+1])
	}
	a.v = v
	return
}

type Any struct {
	v A
}

// Len returns the length of the underlying map/slice.
// if a isn't a map or a slice, Len returns -1.
func (a Any) Len() int {
	switch v := a.v.(type) {
	case []Any:
		return len(v)
	case map[string]Any:
		return len(v)
	case reflect.Value:
		v = reflect.Indirect(v)
		switch v.Kind() {
		case reflect.Map, reflect.Slice, reflect.Array:
			return v.Len()
		}
	}

	return -1
}

func (a Any) ForEach(fn func(key A, value Any) (exit bool)) {
	switch v := a.v.(type) {
	case []Any:
		for i := range v {
			if fn(i, v[i]) {
				return
			}
		}
	case map[string]Any:
		for k := range v {
			if fn(k, v[k]) {
				return
			}
		}
	case reflect.Value:
		v = reflect.Indirect(v)
		switch v.Kind() {
		case reflect.Slice, reflect.Array:
			for i := 0; i < v.Len(); i++ {
				if fn(i, Any{v: v.Index(i)}) {
					return
				}
			}

		case reflect.Map:
			for it := v.MapRange(); it.Next(); {
				if fn(it.Key().Interface(), Value(it.Value())) {
					return
				}
			}
		}
	}
}

// Get will nest for all the given keys, for example:
// Map("key", Slice(1, Map("key", Slice(42)), 3)).Get("key", 1, "key", 0).Int() === 42
func (a Any) Get(keys ...A) (_ Any) {
	for _, key := range keys {
		switch v := a.v.(type) {
		case []Any:
			a = v[key.(int)]
		case map[string]Any:
			a = v[key.(string)]
		default:
			rv := indexReflect(a.v, key)
			if rv == nil {
				return
			}
			a = Value(rv)
		}
	}

	return a
}

func (a Any) Has(key A) bool {
	switch v := a.v.(type) {
	case map[string]Any:
		_, ok := v[key.(string)]
		return ok
	case reflect.Value:
		v = reflect.Indirect(v)
		switch v.Kind() {
		case reflect.Map:
			return v.MapIndex(reflect.ValueOf(key)).IsValid()
		case reflect.Struct:
			return v.FieldByName(key.(string)).IsValid()
		}
	}

	return false
}

func (a Any) Keys() (out []A) {
	switch v := a.v.(type) {
	case map[string]Any:
		out = make([]A, 0, len(v))
		for k := range v {
			out = append(out, k)
		}
	case reflect.Value:
		v = reflect.Indirect(v)
		switch v.Kind() {
		case reflect.Map:
			mk := v.MapKeys()
			out = make([]A, 0, len(mk))
			for i := range mk {
				out = append(out, mk[i].Interface())
			}
		case reflect.Struct:
			t := v.Type()
			out = make([]A, 0, t.NumField())
			for i := 0; i < cap(out); i++ {
				if f := t.Field(i); f.Name != "" {
					out = append(out, f.Name)
				}
			}
		}
	}

	return
}

func (a Any) String() string {
	switch v := a.v.(type) {
	case json.RawMessage:
		return trim(v)
	case string:
		return v
	case reflect.Value:
		if v.Kind() == reflect.String {
			return v.String()
		}
	}

	return ""
}

func (a Any) Int() int64 {
	var s string
	switch v := a.v.(type) {
	case json.RawMessage:
		s = trim(v)
	case json.Number:
		s = string(v)
	case int64:
		return v
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

func (a Any) Uint() uint64 {
	var s string
	switch v := a.v.(type) {
	case json.RawMessage:
		s = trim(v)
	case json.Number:
		s = string(v)
	case uint64:
		return v
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

func (a Any) Float() float64 {
	var s string
	switch v := a.v.(type) {
	case json.RawMessage:
		s = trim(v)
	case json.Number:
		s = string(v)
	case float64:
		return v
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

func (a Any) Bool() bool {
	var s string
	switch v := a.v.(type) {
	case json.RawMessage:
		s = trim(v)
	case string:
		s = v
	case bool:
		return v
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

func (a Any) IsNumber() bool {
	var s string
	switch v := a.v.(type) {
	case json.RawMessage:
		s = trim(v)
	case json.Number:
		s = string(v)
	case int64, uint64, float64:
		return true
	case reflect.Value:
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			return true
		case reflect.String:
			s = v.String()
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

func (a Any) Time(layouts ...string) (t time.Time) {
	if t, _ = a.v.(time.Time); !t.IsZero() {
		return
	}
	const ms = 10000000000
	if n := a.Int(); n > 0 {
		if n > ms {
			n /= 1000 // js date
		}
		return time.Unix(n, 0)
	}

	if s := a.String(); s != "" {
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

func (a *Any) Append(val A) {
	if a.v == nil || a.Type() != "slice" {
		a.v = []Any{}
	}

	switch v := a.v.(type) {
	case []Any:
		a.v = append(v, Value(val))
	case reflect.Value:
		a.v = reflect.Append(v, reflect.ValueOf(val))
	}
}

func (a *Any) SetAt(idx int, val A) {
	if a.v == nil || a.Type() != "slice" {
		a.v = make([]Any, idx+1)
	}

	switch v := a.v.(type) {
	case []Any:
		v[idx].Set(val)
	case reflect.Value:
		v.Index(idx).Set(reflect.ValueOf(val))
	}
}

func (a *Any) SetKeyVal(key, val A) {
	if a.v == nil || a.Type() != "map" {
		a.v = map[string]Any{}
	}

	switch v := a.v.(type) {
	case map[string]Any:
		v[key.(string)] = Value(val)
	case reflect.Value:
		v.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(val))
	}
}

// Set sets the current Any to the given value.
func (a *Any) Set(v A) {
	switch v := v.(type) {
	case nil:
		a.v = nil
	case Any:
		a.v = v.v
	case *Any:
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

func (a *Any) set(rv reflect.Value) {
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
	default:
		a.v = rv
	}
}

func (a Any) Type() string {
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

func (a Any) IsNil() bool {
	return a.v == nil
}

func (a Any) Raw() A {
	return a.v
}

func (a *Any) UnmarshalJSON(b []byte) (err error) {
	switch {
	case len(b) == 0:
		return nil
	case len(b) == 2:
		switch string(b) {
		case "[]":
			a.v = []Any{}
			return
		case "{}":
			a.v = map[string]Any{}
		}
	}

	switch b[0] {
	case '"':
		a.v = string(b[1 : len(b)-1])
	case '[':
		var v []Any
		err = json.Unmarshal(b, &v)
		a.v = v
	case '{':
		var v map[string]Any
		err = json.Unmarshal(b, &v)
		a.v = v
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-', '+', '.':
		a.v = json.Number(b)
	case 't', 'T':
		a.v = true
	case 'f', 'F':
		a.v = false
	default:
		var v A
		err = json.Unmarshal(b, &v)
		a.Set(v)
	}

	if err != nil {
		a.v = nil
	}
	return
}

func (a Any) MarshalJSON() ([]byte, error) {
	if rv, ok := a.v.(reflect.Value); ok {
		return json.Marshal(rv.Interface())
	}
	return json.Marshal(a.v)
}

func (a Any) Format(s fmt.State, c rune) {
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

func indexReflect(v A, keyOrIndex A) A {
	var rv, ov reflect.Value

	if v, ok := v.(reflect.Value); ok {
		rv = reflect.Indirect(v)
	} else {
		rv = reflect.Indirect(reflect.ValueOf(v))
	}

	switch rv.Kind() {
	case reflect.Map:
		ov = rv.MapIndex(reflect.ValueOf(keyOrIndex))
	case reflect.Array, reflect.Slice:
		i, _ := keyOrIndex.(int)
		ov = rv.Index(i)
	case reflect.Struct:
		s, _ := keyOrIndex.(string)
		if f := rv.FieldByName(s); f.IsValid() {
			ov = f
		}
	}

	if ov.Kind() != reflect.Invalid {
		return ov
	}

	return nil
}
