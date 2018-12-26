package bencode

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestEncode(t *testing.T) {
	type encodeTestCase struct {
		in  interface{}
		out string
		err bool
	}

	type eT struct {
		A string
		X string `bencode:"D"`
		Y string `bencode:"B"`
		Z string `bencode:"C"`
	}

	type sortProblem struct {
		A string
		B string `bencode:","`
	}

	type issue18Sub struct {
		Name string
	}

	type issue18 struct {
		T *issue18Sub
	}

	type Embedded struct {
		B string
	}

	type issue22 struct {
		Time myTimeType `bencode:"t"`
		Foo  myBoolType `bencode:"f"`
	}

	type issue22WithErrorChild struct {
		Name  string           `bencode:"n"`
		Error errorMarshalType `bencode:"e"`
	}

	type issue26 struct {
		Answer int64          `bencode:"a"`
		Foo    myBoolTextType `bencode:"f"`
		Name   string         `bencode:"n"`
	}

	type issue26WithErrorChild struct {
		Name  string               `bencode:"n"`
		Error errorTextMarshalType `bencode:"e"`
	}

	type issue28 struct {
		X    string        `bencode:"x"`
		Time *myTimeType   `bencode:"t"`
		Foo  myBoolPtrType `bencode:"f"`
		Y    string        `bencode:"y"`
	}

	now := time.Now()

	var encodeCases = []encodeTestCase{
		// integers
		{10, `i10e`, false},
		{-10, `i-10e`, false},
		{0, `i0e`, false},
		{int(10), `i10e`, false},
		{int8(10), `i10e`, false},
		{int16(10), `i10e`, false},
		{int32(10), `i10e`, false},
		{int64(10), `i10e`, false},
		{uint(10), `i10e`, false},
		{uint8(10), `i10e`, false},
		{uint16(10), `i10e`, false},
		{uint32(10), `i10e`, false},
		{uint64(10), `i10e`, false},
		{(*int)(nil), ``, false},

		// ptr-to-integer
		{func() *int {
			i := 42
			return &i
		}(), `i42e`, false},

		// strings
		{"foo", `3:foo`, false},
		{"barbb", `5:barbb`, false},
		{"", `0:`, false},
		{(*string)(nil), ``, false},

		// ptr-to-string
		{func() *string {
			str := "foo"
			return &str
		}(), `3:foo`, false},

		// lists
		{[]interface{}{"foo", 20}, `l3:fooi20ee`, false},
		{[]interface{}{90, 20}, `li90ei20ee`, false},
		{[]interface{}{[]interface{}{"foo", "bar"}, 20}, `ll3:foo3:barei20ee`, false},
		{[]map[string]int{
			{"a": 0, "b": 1},
			{"c": 2, "d": 3},
		}, `ld1:ai0e1:bi1eed1:ci2e1:di3eee`, false},
		{[][]byte{
			[]byte{'0', '2', '4', '6', '8'},
			[]byte{'a', 'c', 'e'},
		}, `l5:024683:acee`, false},
		{(*[]interface{})(nil), ``, false},

		// boolean
		{true, "i1e", false},
		{false, "i0e", false},
		{(*bool)(nil), ``, false},

		// dicts
		{map[string]interface{}{
			"a": "foo",
			"c": "bar",
			"b": "tes",
		}, `d1:a3:foo1:b3:tes1:c3:bare`, false},
		{eT{"foo", "bar", "far", "boo"}, `d1:A3:foo1:B3:far1:C3:boo1:D3:bare`, false},
		{map[string][]int{
			"a": {0, 1},
			"b": {2, 3},
		}, `d1:ali0ei1ee1:bli2ei3eee`, false},
		{struct{ A, b int }{1, 2}, "d1:Ai1ee", false},
		{(*struct{ A int })(nil), ``, false},

		// raw
		{RawMessage(`i5e`), `i5e`, false},
		{[]RawMessage{
			RawMessage(`i5e`),
			RawMessage(`5:hello`),
			RawMessage(`ldededee`),
		}, `li5e5:helloldededeee`, false},
		{map[string]RawMessage{
			"a": RawMessage(`i5e`),
			"b": RawMessage(`5:hello`),
			"c": RawMessage(`ldededee`),
		}, `d1:ai5e1:b5:hello1:cldededeee`, false},

		// problem sorting
		{sortProblem{A: "foo", B: "bar"}, `d1:A3:foo1:B3:bare`, false},

		// nil values dropped from maps and structs
		{map[string]*int{"a": nil}, `de`, false},
		{struct{ A *int }{nil}, `de`, false},
		{issue18{}, `de`, false},
		{map[string]interface{}{"a": nil}, `de`, false},
		{struct{ A interface{} }{nil}, `de`, false},

		// embedded structs
		{struct {
			A string
			Embedded
		}{"foo", Embedded{"bar"}}, `d1:A3:foo1:B3:bare`, false},
		{struct {
			A        string
			Embedded `bencode:"C"`
		}{"foo", Embedded{"bar"}}, `d1:A3:foo1:Cd1:B3:baree`, false},

		// embedded structs order issue #20
		{struct {
			Embedded
			A string
		}{Embedded{"bar"}, "foo"}, `d1:A3:foo1:B3:bare`, false},

		// types which implement the Marshal interface will
		// be marshalled using this interface
		{myBoolType(true), `1:y`, false},
		{myBoolType(false), `1:n`, false},
		{myTimeType{now}, fmt.Sprintf("i%de", now.Unix()), false},
		{errorMarshalType{}, "", true},

		// pointers to types which implement the Marshal interface will
		// be marshalled using this interface
		{func() *myBoolType {
			b := myBoolType(true)
			return &b
		}(), `1:y`, false},
		{func() *myTimeType {
			t := myTimeType{now}
			return &t
		}(), fmt.Sprintf("i%de", now.Unix()), false},
		{func() *errorMarshalType {
			e := errorMarshalType{}
			return &e
		}(), "", true},

		// nil-pointers to types which implement the Marshal interface will be ignored
		{(*myBoolType)(nil), "", false},
		{(*myTimeType)(nil), "", false},
		{(*errorMarshalType)(nil), "", false},

		// ptr-types which implements the Marshal interface will
		// be marshalled using this interface
		{func() *myBoolPtrType {
			b := myBoolPtrType(true)
			return &b
		}(), `1:y`, false},
		{func() *myBoolPtrType {
			b := myBoolPtrType(false)
			return &b
		}(), `1:n`, false},
		{(*myBoolPtrType)(nil), ``, false},

		// structures can also have children which support
		// the Marshal interface
		{
			issue22{Time: myTimeType{now}, Foo: myBoolType(true)},
			fmt.Sprintf("d1:f1:y1:ti%dee", now.Unix()),
			false,
		},
		{ // an error will be returned if a child can't be marshalled
			issue22WithErrorChild{Name: "Foo", Error: errorMarshalType{}},
			"", true,
		},
		// structures passed by reference which have children that support
		// the (Text)Marshal interface (by value or by reference),
		// will be marshaled using that interface
		{
			&issue22{Time: myTimeType{now}, Foo: myBoolType(true)},
			fmt.Sprintf("d1:f1:y1:ti%dee", now.Unix()),
			false,
		},
		{ // an error will be returned if a child can't be marshalled
			&issue22WithErrorChild{Name: "Foo", Error: errorMarshalType{}},
			"", true,
		},

		// types which implement the TextMarshal interface will
		// be marshalled into a bencode string value using this interface
		{myBoolTextType(true), `1:y`, false},
		{myBoolTextType(false), `1:n`, false},
		{errorTextMarshalType{}, "", true},

		// structures can also have children which support
		// the TextMarshal interface
		{
			issue26{Answer: 42, Foo: myBoolTextType(true), Name: "Nova"},
			`d1:ai42e1:f1:y1:n4:Novae`,
			false,
		},
		{ // an error will be returned if a child TextMarshaler returns an error
			issue26WithErrorChild{Name: "Foo", Error: errorTextMarshalType{}},
			"", true,
		},

		// ptr types which are used as value types,
		// but which ptr version implement the Marshaler/TextMarshaler interface,
		// will still get marshalling using this interface, when possible
		{
			&issue28{X: "x", Time: &myTimeType{now}, Foo: myBoolPtrType(true), Y: "y"},
			fmt.Sprintf(`d1:f1:y1:ti%de1:x1:x1:y1:ye`, now.Unix()),
			false,
		},
	}

	for i, tt := range encodeCases {
		t.Logf("%d: %#v", i, tt.in)
		data, err := EncodeString(tt.in)
		if !tt.err && err != nil {
			t.Errorf("#%d: Unexpected err: %v", i, err)
			continue
		}
		if tt.err && err == nil {
			t.Errorf("#%d: Expected err is nil", i)
			continue
		}
		if tt.out != data {
			t.Errorf("#%d: Val: %q != %q", i, data, tt.out)
		}
	}
}

type myBoolPtrType bool

// MarshalBencode implements Marshaler.MarshalBencode
func (mbt *myBoolPtrType) MarshalBencode() ([]byte, error) {
	var c string
	if *mbt {
		c = "y"
	} else {
		c = "n"
	}

	return EncodeBytes(c)
}

// UnmarshalBencode implements Unmarshaler.UnmarshalBencode
func (mbt *myBoolPtrType) UnmarshalBencode(b []byte) error {
	var str string
	err := DecodeBytes(b, &str)
	if err != nil {
		return err
	}

	switch str {
	case "y":
		*mbt = true
	case "n":
		*mbt = false
	default:
		err = errors.New("invalid myBoolType")
	}

	return err
}

func TestEncodeOmit(t *testing.T) {
	type encodeTestCase struct {
		in  interface{}
		out string
		err bool
	}

	type eT struct {
		A string `bencode:",omitempty"`
		B int    `bencode:",omitempty"`
		C *int   `bencode:",omitempty"`
	}

	var encodeCases = []encodeTestCase{
		{eT{}, `de`, false},
		{eT{A: "a"}, `d1:A1:ae`, false},
		{eT{B: 5}, `d1:Bi5ee`, false},
		{eT{C: new(int)}, `d1:Ci0ee`, false},
	}

	for i, tt := range encodeCases {
		data, err := EncodeString(tt.in)
		if !tt.err && err != nil {
			t.Errorf("#%d: Unexpected err: %v", i, err)
			continue
		}
		if tt.err && err == nil {
			t.Errorf("#%d: Expected err is nil", i)
			continue
		}
		if tt.out != data {
			t.Errorf("#%d: Val: %q != %q", i, data, tt.out)
		}
	}
}
