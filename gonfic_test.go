package gonfic

import (
	"testing"
	"fmt"
	"time"
)

type testValue struct {
	B bool
	S string
	I int
	U uint
	F float32
	M map[string]string
	A []string
	D time.Duration
}

type testConfig struct {
	Values map[string]testValue
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

func test(t *testing.T, buf string, ext string) {
	c := NewConfig()
	err := c.AddSource(NewBufSource([]byte(buf), ext))
	if err != nil {
		t.Errorf("unable to add buf (%s) source: %s", ext, err)
	}
	err = c.AddSource(NewEnvSource("my"))
	if err != nil {
		t.Errorf("unable to add env source: %s", err)
	}
	v := new(testConfig)
	err = c.Unmarshal(v)
	if err != nil {
		t.Errorf("unable to unmarshal: %s", err)
	}
	fmt.Printf("%#v", v)
}
