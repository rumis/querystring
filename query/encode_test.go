package query

import (
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
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

	t.Error(str)

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

	t.Error(str)

}
