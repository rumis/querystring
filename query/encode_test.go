package query

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestMapEncode(t *testing.T) {

	m1 := map[string]interface{}{
		"name": "murong",
		"age": map[string]interface{}{
			"a1": 1,
			"a2": "2",
			"sex": map[string]interface{}{
				"s3": 3,
				"s5": "ssss",
			},
			"time": time.Now(),
		},
	}

	v, err := Values(m1)
	if err != nil {
		t.Fatal(err)
	}

	str := v.Encode()

	str = strings.ReplaceAll(str, "%5B", "[")
	str = strings.ReplaceAll(str, "%5D", "]")

	// t.Error(str)

}

type Level struct {
	Level int `qs:"level"`
}

func (l Level) EncodeValues(scope string, v *url.Values) error {
	v.Add("lt", "t-"+strconv.Itoa(l.Level))
	return nil
}

func TestStructEncode(t *testing.T) {

	s1 := struct {
		Level
		ID       int `qs:"-"`
		Name     string
		Age      int       `qs:"age,omitempty"`
		Sex      string    `qs:"sex"`
		Subject  []string  `qs:"subject"`
		Birthday time.Time `qs:"birthday"`
	}{
		Level: Level{
			Level: 99,
		},
		ID:   2,
		Name: "zhangsan",
		Age:  1,
		Sex:  "man",
		Subject: []string{
			"math", "english", "chinese",
		},
		Birthday: time.Now(),
	}

	v, err := Values(s1)
	if err != nil {
		t.Fatal(err)
	}

	str := v.Encode()

	str = strings.ReplaceAll(str, "%5B", "[")
	str = strings.ReplaceAll(str, "%5D", "]")

	// t.Error(str)

}

// test that Values(input) matches want.  If not, report an error on t.
func testValue(t *testing.T, input interface{}, want url.Values) {
	v, err := Values(input)
	if err != nil {
		t.Errorf("Values(%q) returned error: %v", input, err)
	}
	if diff := cmp.Diff(want, v); diff != "" {
		t.Errorf("Values(%#v) mismatch:\n%s", input, diff)
	}
}

func TestValues_BasicTypes(t *testing.T) {
	tests := []struct {
		input interface{}
		want  url.Values
	}{
		// zero values
		{struct{ V string }{}, url.Values{"V": {""}}},
		{struct{ V int }{}, url.Values{"V": {"0"}}},
		{struct{ V uint }{}, url.Values{"V": {"0"}}},
		{struct{ V float32 }{}, url.Values{"V": {"0"}}},
		{struct{ V bool }{}, url.Values{"V": {"false"}}},

		// simple non-zero values
		{struct{ V string }{"v"}, url.Values{"V": {"v"}}},
		{struct{ V int }{1}, url.Values{"V": {"1"}}},
		{struct{ V uint }{1}, url.Values{"V": {"1"}}},
		{struct{ V float32 }{0.1}, url.Values{"V": {"0.1"}}},
		{struct{ V bool }{true}, url.Values{"V": {"true"}}},

		// bool-specific options
		{
			struct {
				V bool `qs:""`
			}{false},
			url.Values{"V": {"false"}},
		},
		{
			struct {
				V bool `qs:""`
			}{true},
			url.Values{"V": {"true"}},
		},

		// time values
		{
			struct {
				V time.Time
			}{time.Date(2000, 1, 1, 12, 34, 56, 0, time.UTC)},
			url.Values{"V": {"2000-01-01 12:34:56"}},
		},
	}

	for _, tt := range tests {
		testValue(t, tt.input, tt.want)
	}
}

func TestValues_Pointers(t *testing.T) {
	str := "s"
	strPtr := &str

	tests := []struct {
		input interface{}
		want  url.Values
	}{
		// nil pointers (zero values)
		{struct{ V *string }{}, url.Values{}},
		{struct{ V *int }{}, url.Values{}},

		// non-zero pointer values
		{struct{ V *string }{&str}, url.Values{"V": {"s"}}},
		{struct{ V **string }{&strPtr}, url.Values{"V": {"s"}}},

		// slices of pointer values
		{struct{ V []*string }{}, url.Values{}},
		{struct{ V []*string }{[]*string{&str, &str}}, url.Values{"V[]": {"s", "s"}}},

		// pointer to slice
		{struct{ V *[]string }{}, url.Values{}},
		{struct{ V *[]string }{&[]string{"a", "b"}}, url.Values{"V[]": {"a", "b"}}},

		// pointer values for the input struct itself
		{(*struct{})(nil), url.Values{}},
		{&struct{}{}, url.Values{}},
		{&struct{ V string }{}, url.Values{"V": {""}}},
		{&struct{ V string }{"v"}, url.Values{"V": {"v"}}},
	}

	for _, tt := range tests {
		testValue(t, tt.input, tt.want)
	}
}

func TestValues_Slices(t *testing.T) {
	tests := []struct {
		input interface{}
		want  url.Values
	}{
		// slices of strings
		{
			struct{ V []string }{},
			url.Values{},
		},
		{
			struct{ V []string }{[]string{}},
			url.Values{},
		},
		{
			struct{ V []string }{[]string{""}},
			url.Values{"V[]": {""}},
		},
		{
			struct{ V []string }{[]string{"a", "b"}},
			url.Values{"V[]": {"a", "b"}},
		},
		// arrays of strings
		{
			struct{ V [2]string }{},
			url.Values{"V[]": {"", ""}},
		},
		{
			struct{ V [2]string }{[2]string{"a", "b"}},
			url.Values{"V[]": {"a", "b"}},
		},
	}

	for _, tt := range tests {
		testValue(t, tt.input, tt.want)
	}
}

func TestValues_NestedTypes(t *testing.T) {
	type SubNested struct {
		Value string `qs:"value"`
	}

	type Nested struct {
		A   SubNested  `qs:"a"`
		B   *SubNested `qs:"b"`
		Ptr *SubNested `qs:"ptr,omitempty"`
	}

	tests := []struct {
		input interface{}
		want  url.Values
	}{
		{
			struct {
				Nest Nested `qs:"nest"`
			}{
				Nested{
					A: SubNested{
						Value: "v",
					},
				},
			},
			url.Values{
				"nest[a][value]": {"v"},
			},
		},
		{
			struct {
				Nest Nested `qs:"nest"`
			}{
				Nested{
					Ptr: &SubNested{
						Value: "v",
					},
				},
			},
			url.Values{
				"nest[a][value]":   {""},
				"nest[ptr][value]": {"v"},
			},
		},
		{
			nil,
			url.Values{},
		},
	}

	for _, tt := range tests {
		testValue(t, tt.input, tt.want)
	}
}

func TestValues_OmitEmpty(t *testing.T) {
	str := ""

	tests := []struct {
		input interface{}
		want  url.Values
	}{
		{struct{ v string }{}, url.Values{}}, // non-exported field
		{
			struct {
				V string `qs:",omitempty"`
			}{},
			url.Values{},
		},
		{
			struct {
				V string `qs:"-"`
			}{},
			url.Values{},
		},
		{
			struct {
				V string `qs:"omitempty"` // actually named omitempty
			}{},
			url.Values{"omitempty": {""}},
		},
		{
			// include value for a non-nil pointer to an empty value
			struct {
				V *string `qs:",omitempty"`
			}{&str},
			url.Values{"V": {""}},
		},
	}

	for _, tt := range tests {
		testValue(t, tt.input, tt.want)
	}
}

func TestValues_EmbeddedStructs(t *testing.T) {
	type Inner struct {
		V string
	}
	type Outer struct {
		Inner
	}
	type OuterPtr struct {
		*Inner
	}
	type Mixed struct {
		Inner
		V string
	}
	type unexported struct {
		Inner
		V string
	}
	type Exported struct {
		unexported
	}

	tests := []struct {
		input interface{}
		want  url.Values
	}{
		{
			Outer{Inner{V: "a"}},
			url.Values{"V": {"a"}},
		},
		{
			OuterPtr{&Inner{V: "a"}},
			url.Values{"V": {"a"}},
		},
		{
			Mixed{Inner: Inner{V: "a"}, V: "b"},
			url.Values{"V": {"a", "b"}},
		},
		{
			// values from unexported embed are still included
			Exported{
				unexported{
					Inner: Inner{V: "bar"},
					V:     "foo",
				},
			},
			url.Values{"V": {"bar", "foo"}},
		},
	}

	for _, tt := range tests {
		testValue(t, tt.input, tt.want)
	}
}

func TestValues_InvalidInput(t *testing.T) {
	_, err := Values("")
	if err == nil {
		t.Errorf("expected Values() to return an error on invalid input")
	}
}

// customEncodedStrings is a slice of strings with a custom URL encoding
type customEncodedStrings []string

// EncodeValues using key name of the form "{key}.N" where N increments with
// each value.  A value of "err" will return an error.
func (m customEncodedStrings) EncodeValues(key string, v *url.Values) error {
	for i, arg := range m {
		if arg == "err" {
			return errors.New("encoding error")
		}
		v.Set(fmt.Sprintf("%s.%d", key, i), arg)
	}
	return nil
}

func TestValues_CustomEncodingSlice(t *testing.T) {
	tests := []struct {
		input interface{}
		want  url.Values
	}{
		{
			struct {
				V customEncodedStrings `qs:"v"`
			}{},
			url.Values{},
		},
		{
			struct {
				V customEncodedStrings `qs:"v"`
			}{[]string{"a", "b"}},
			url.Values{"v.0": {"a"}, "v.1": {"b"}},
		},

		// pointers to custom encoded types
		{
			struct {
				V *customEncodedStrings `qs:"v"`
			}{},
			url.Values{},
		},
		{
			struct {
				V *customEncodedStrings `qs:"v"`
			}{(*customEncodedStrings)(&[]string{"a", "b"})},
			url.Values{"v.0": {"a"}, "v.1": {"b"}},
		},
	}

	for _, tt := range tests {
		testValue(t, tt.input, tt.want)
	}
}

// One of the few ways reflectValues will return an error is if a custom
// encoder returns an error.  Test all of the various ways that can happen.
func TestValues_CustomEncoding_Error(t *testing.T) {
	type st struct {
		V customEncodedStrings
	}
	tests := []struct {
		input interface{}
	}{
		{
			st{[]string{"err"}},
		},
		{ // struct field
			struct{ S st }{st{[]string{"err"}}},
		},
		{ // embedded struct
			struct{ st }{st{[]string{"err"}}},
		},
	}
	for _, tt := range tests {
		_, err := Values(tt.input)
		if err == nil {
			t.Errorf("Values(%q) did not return expected encoding error", tt.input)
		}
	}
}

// customEncodedInt is an int with a custom URL encoding
type customEncodedInt int

// EncodeValues encodes values with leading underscores
func (m customEncodedInt) EncodeValues(key string, v *url.Values) error {
	v.Set(key, fmt.Sprintf("_%d", m))
	return nil
}

func TestValues_CustomEncodingInt(t *testing.T) {
	var zero customEncodedInt = 0
	var one customEncodedInt = 1
	tests := []struct {
		input interface{}
		want  url.Values
	}{
		{
			struct {
				V customEncodedInt `qs:"v"`
			}{},
			url.Values{"v": {"_0"}},
		},
		{
			struct {
				V customEncodedInt `qs:"v,omitempty"`
			}{zero},
			url.Values{},
		},
		{
			struct {
				V customEncodedInt `qs:"v"`
			}{one},
			url.Values{"v": {"_1"}},
		},

		{
			struct {
				V *customEncodedInt `qs:"v,omitempty"`
			}{},
			url.Values{},
		},
		{
			struct {
				V *customEncodedInt `qs:"v,omitempty"`
			}{&zero},
			url.Values{"v": {"_0"}},
		},
		{
			struct {
				V *customEncodedInt `qs:"v"`
			}{&one},
			url.Values{"v": {"_1"}},
		},
	}

	for _, tt := range tests {
		testValue(t, tt.input, tt.want)
	}
}

// customEncodedInt is an int with a custom URL encoding defined on its pointer
// value.
type customEncodedIntPtr int

// EncodeValues encodes a 0 as false, 1 as true, and nil as unknown.  All other
// values cause an error.
func (m *customEncodedIntPtr) EncodeValues(key string, v *url.Values) error {
	if m == nil {
		v.Set(key, "undefined")
	} else {
		v.Set(key, fmt.Sprintf("_%d", *m))
	}
	return nil
}

// Test behavior when encoding is defined for a pointer of a custom type.
// Custom type should be able to encode values for nil pointers.
func TestValues_CustomEncodingPointer(t *testing.T) {
	var zero customEncodedIntPtr = 0
	var one customEncodedIntPtr = 1
	tests := []struct {
		input interface{}
		want  url.Values
	}{
		// non-pointer values do not get the custom encoding because
		// they don't implement the encoder interface.
		{
			struct {
				V customEncodedIntPtr `qs:"v"`
			}{},
			url.Values{"v": {"0"}},
		},
		{
			struct {
				V customEncodedIntPtr `qs:"v,omitempty"`
			}{},
			url.Values{},
		},
		{
			struct {
				V customEncodedIntPtr `qs:"v"`
			}{one},
			url.Values{"v": {"1"}},
		},

		// pointers to custom encoded types.
		{
			struct {
				V *customEncodedIntPtr `qs:"v"`
			}{},
			url.Values{},
		},
		{
			struct {
				V *customEncodedIntPtr `qs:"v,omitempty"`
			}{},
			url.Values{},
		},
		{
			struct {
				V *customEncodedIntPtr `qs:"v"`
			}{&zero},
			url.Values{"v": {"0"}},
		},
		{
			struct {
				V *customEncodedIntPtr `qs:"v,omitempty"`
			}{&zero},
			url.Values{"v": {"0"}},
		},
		{
			struct {
				V *customEncodedIntPtr `qs:"v"`
			}{&one},
			url.Values{"v": {"1"}},
		},
	}

	for _, tt := range tests {
		testValue(t, tt.input, tt.want)
	}
}

func TestIsEmptyValue(t *testing.T) {
	str := "string"
	tests := []struct {
		value interface{}
		empty bool
	}{
		// slices, arrays, and maps
		{[]int{}, true},
		{[]int{0}, false},
		{[0]int{}, true},
		{[3]int{}, false},
		{[3]int{1}, false},
		{map[string]string{}, true},
		{map[string]string{"a": "b"}, false},

		// strings
		{"", true},
		{" ", false},
		{"a", false},

		// bool
		{true, false},
		{false, true},

		// ints of various types
		{(int)(0), true}, {(int)(1), false}, {(int)(-1), false},
		{(int8)(0), true}, {(int8)(1), false}, {(int8)(-1), false},
		{(int16)(0), true}, {(int16)(1), false}, {(int16)(-1), false},
		{(int32)(0), true}, {(int32)(1), false}, {(int32)(-1), false},
		{(int64)(0), true}, {(int64)(1), false}, {(int64)(-1), false},
		{(uint)(0), true}, {(uint)(1), false},
		{(uint8)(0), true}, {(uint8)(1), false},
		{(uint16)(0), true}, {(uint16)(1), false},
		{(uint32)(0), true}, {(uint32)(1), false},
		{(uint64)(0), true}, {(uint64)(1), false},

		// floats
		{(float32)(0), true}, {(float32)(0.0), true}, {(float32)(0.1), false},
		{(float64)(0), true}, {(float64)(0.0), true}, {(float64)(0.1), false},

		// pointers
		{(*int)(nil), true},
		{new([]int), false},
		{&str, false},

		// time
		{time.Time{}, true},
		{time.Now(), false},

		// unknown type - always false unless a nil pointer, which are always empty.
		{(*struct{ int })(nil), true},
		{struct{ int }{}, false},
		{struct{ int }{0}, false},
		{struct{ int }{1}, false},
	}

	for _, tt := range tests {
		got := isEmptyValue(reflect.ValueOf(tt.value))
		want := tt.empty
		if got != want {
			t.Errorf("isEmptyValue(%v) returned %t; want %t", tt.value, got, want)
		}
	}
}

func TestParseTag(t *testing.T) {
	name, opts := parseTag("field,foobar,foo")
	if name != "field" {
		t.Fatalf("name = %q, want field", name)
	}
	for _, tt := range []struct {
		opt  string
		want bool
	}{
		{"foobar", true},
		{"foo", true},
		{"bar", false},
		{"field", false},
	} {
		if opts.Contains(tt.opt) != tt.want {
			t.Errorf("Contains(%q) = %v", tt.opt, !tt.want)
		}
	}
}
