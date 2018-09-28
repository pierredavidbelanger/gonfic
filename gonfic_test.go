package gonfic

import (
	"fmt"
	"testing"
	"time"
)

type testValue struct {
	B bool              `json:"b,omitempty"`
	S string            `json:"s,omitempty"`
	I int               `json:"i,omitempty"`
	U uint              `json:"u,omitempty"`
	F float32           `json:"f,omitempty"`
	M map[string]string `json:"m,omitempty"`
	A []string          `json:"a,omitempty"`
	D time.Duration     `json:"d,omitempty"`
}

type testConfig struct {
	Values map[string]*testValue `json:"values"`
}

func TestJSON(t *testing.T) {
	buf := `{
  "values": {
    "v1": {
      "b": true,
      "s": "hello world",
      "i": -42,
      "u": 42,
      "f": 3.1416,
      "m": {
        "k1": "v1"
      },
      "a": [
        "e1"
      ],
      "d": "1m"
    }
  }
}`
	test(t, buf, "json")
}

func TestYAML(t *testing.T) {
	buf := `values:
  v1:
    b: true
    s: hello world
    i: -42
    u: 42
    f: 3.1416
    m:
      k1: v1
    a:
      - e1
    d: 1m`
	test(t, buf, "yaml")
}

func TestStruct(t *testing.T) {
	in := testConfig{Values: map[string]*testValue{"v1": {S: "hello world"}}}
	c := NewConfig()
	c.AddSource(NewStructSource("", in))
	out := testConfig{}
	c.Unmarshal("", &out)
	fmt.Printf("%#v", out)
}

func test(t *testing.T, buf string, ext string) {
	c := NewConfig()
	err := c.AddSource(NewBufSource([]byte(buf), ext))
	if err != nil {
		t.Errorf("unable to add buf (%s) source: %s", ext, err)
	}
	if err != nil {
		t.Errorf("unable to add env source: %s", err)
	}
	v := new(testConfig)
	err = c.Unmarshal("", v)
	if err != nil {
		t.Errorf("unable to unmarshal: %s", err)
	}
	fmt.Printf("%#v", v)
}
