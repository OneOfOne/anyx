package anyx

import "testing"

type S struct {
	V int
}

func TestMapperAndSlicer(t *testing.T) {
	ss := []S{{1}, {2}, {3}}
	v := SliceOf(ss)
	if v.Len() != 3 {
		t.Fatal("bad len")
	}
	if v.Get(1).v != ss[1] {
		t.Fatal("bad get")
	}

	if !v.Has(S{2}) {
		t.Fatal("bad has")
	}

	v.SetAt(1, S{4})

	if !v.Has(S{4}) {
		t.Fatal("bad has 2")
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	ss := []S{{1}, {2}, {3}}
	v := SliceOf(ss)
	b, err := v.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(b))
}
