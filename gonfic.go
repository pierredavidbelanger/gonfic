package gonfic

import (
	"encoding/json"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"io/ioutil"
	"os"
	"strings"
)

type Config interface {
	AddSource(s Source) error
	ToFlatMap() map[string]interface{}
	ToMap() map[string]interface{}
	Unmarshal(v interface{}) error
}

type config struct {
	flat map[string]interface{}
}

func (c *config) AddSource(s Source) error {
	flat, err := s.Override(c.flat)
	if err != nil {
		return err
	}
	c.flat = flat
	return nil
}

func (c *config) ToFlatMap() map[string]interface{} {
	return c.flat
}

func (c *config) ToMap() map[string]interface{} {
	return Unflatten(c.ToFlatMap(), DotSlicer)
}

func (c *config) Unmarshal(v interface{}) error {
	return mapstructure.Decode(c.ToMap(), v)
}

func NewConfig() Config {
	c := &config{}
	c.flat = make(map[string]interface{})
	return c
}

type Source interface {
	Override(map[string]interface{}) (map[string]interface{}, error)
}

type envSource struct {
	prefix string
}

func (s *envSource) Override(config map[string]interface{}) (map[string]interface{}, error) {
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		key, value := pair[0], pair[1]
		key = strings.ToLower(key)
		key = strings.Replace(key, "_", ".", -1)
		if !strings.HasPrefix(key, s.prefix+".") {
			continue
		}
		key = strings.TrimPrefix(key, s.prefix+".")
		config[key] = value
	}
	return config, nil
}

func NewEnvSource(prefix string) Source {
	return &envSource{prefix: prefix}
}

type fileSource struct {
	path string
}

func (s *fileSource) Override(config map[string]interface{}) (map[string]interface{}, error) {
	buf, err := ioutil.ReadFile(s.path)
	if err != nil {
		return config, fmt.Errorf("cannot read file %s: %s", s.path, err)
	}
	m := make(map[string]interface{})
	err = json.Unmarshal(buf, &m)
	if err != nil {
		return config, fmt.Errorf("cannot unmarshall JSON file %s: %s", s.path, err)
	}
	fm := Flatten(m, DotJoiner)
	for key, value := range fm {
		config[key] = value
	}
	return config, nil
}

func NewFileSource(path string) Source {
	return &fileSource{path: path}
}

var DotJoiner = func(a []string) string { return strings.Join(a, ".") }

var DotSlicer = func(s string) []string { return strings.Split(s, ".") }

func Unflatten(flatmap map[string]interface{}, slicer func(string) []string) map[string]interface{} {
	var unflatmap = make(map[string]interface{})
	for flatkey, value := range flatmap {
		keys := slicer(flatkey)
		subunflatmap := unflatmap
		for _, key := range keys[:len(keys)-1] {
			node, ok := subunflatmap[key]
			if !ok {
				node = make(map[string]interface{})
				subunflatmap[key] = node
			}
			subunflatmap = node.(map[string]interface{})
		}
		subunflatmap[keys[len(keys)-1]] = value
	}
	return unflatmap
}

func Flatten(unflatmap map[string]interface{}, joiner func([]string) string) map[string]interface{} {
	var flatmap = make(map[string]interface{})
	flattenrec(unflatmap, []string{}, func(keys []string, value interface{}) {
		flatmap[joiner(keys)] = value
	})
	return flatmap
}

func flattenrec(unflatmap map[string]interface{}, keys []string, adder func(keys []string, value interface{})) {
	for key, value := range unflatmap {
		subkeys := append(keys, key)
		if subunflatmap, ok := value.(map[string]interface{}); ok {
			flattenrec(subunflatmap, subkeys, adder)
		} else {
			adder(subkeys, value)
		}
	}
}
