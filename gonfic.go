package gonfic

import (
	"encoding/json"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"github.com/ghodss/yaml"
	"reflect"
	"time"
)

// Source is the interface implemented by types
// that can override an existing flat map of keys and values
// with values from a custom source.
type Source interface {
	Override(map[string]interface{}) (map[string]interface{}, error)
}

// Config holds keys and values from different sources and
// can transform them into hierarchical map, flat map or
// unmarshal them unto a struct.
type Config struct {
	flat map[string]interface{}
}

func NewConfig() *Config {
	c := &Config{}
	c.flat = make(map[string]interface{})
	return c
}

// AddSource is used to load keys and values into the config.
func (c *Config) AddSource(s Source) error {
	flat, err := s.Override(c.flat)
	if err != nil {
		return err
	}
	c.flat = flat
	return nil
}

// ToFlatMap returns a flat map of the keys and values in the config.
func (c *Config) ToFlatMap() map[string]interface{} {
	return c.flat
}

// ToHierarchicalMap returns the keys and values in the config in a hierarchical map,
// so when repeating config path are agglomerated (think JSON).
func (c *Config) ToHierarchicalMap() map[string]interface{} {
	return unflatten(c.ToFlatMap(), dotSlicer)
}

// Unmarshal the keys and values as an hierarchical map
// and stores the result in the value pointed to by v.
func (c *Config) Unmarshal(v interface{}) error {
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: decodeHook,
		WeaklyTypedInput: true,
		Result: v,
	})
	if err != nil {
		return err
	}
	return dec.Decode(c.ToHierarchicalMap())
}

func decodeHook(srcType  reflect.Type, dstType reflect.Type, v interface{}) (interface{}, error) {
	// not sure this is the way to go
	if srcType.Kind() == reflect.String && dstType.String() == "time.Duration" {
		return time.ParseDuration(v.(string))
	}
	return v, nil
}

type structSource struct {
	s interface{}
}

func NewStruckSource(s interface{}) Source {
	return &structSource{s: s}
}

func (s *structSource) Override(config map[string]interface{}) (map[string]interface{}, error) {
	buf, err := json.Marshal(s.s)
	if err != nil {
		return config, err
	}
	bufSource := NewBufSource(buf, "json")
	return bufSource.Override(config)
}

type envSource struct {
	prefix string
}

func NewEnvSource(prefix string) Source {
	return &envSource{prefix: strings.ToLower(prefix)}
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

type fileSource struct {
	path string
}

func NewFileSource(path string) Source {
	return &fileSource{path: path}
}

func (s *fileSource) Override(config map[string]interface{}) (map[string]interface{}, error) {
	buf, err := ioutil.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("cannot read: %s", err)
	}
	ext := filepath.Ext(s.path)
	if strings.HasPrefix(ext, ".") {
		ext = strings.TrimPrefix(ext, ".")
	}
	bufSource := NewBufSource(buf, ext)
	return bufSource.Override(config)
}

type bufSource struct {
	buf []byte
	ext string
}

func NewBufSource(buf []byte, ext string) Source {
	return &bufSource{buf: buf, ext: strings.ToLower(ext)}
}

func (s *bufSource) Override(config map[string]interface{}) (map[string]interface{}, error) {
	fm, err := readBuf(s.buf, s.ext)
	if err != nil {
		return config, err
	}
	for key, value := range fm {
		config[key] = value
	}
	return config, nil
}

func readBuf(buf []byte, ext string) (map[string]interface{}, error) {
	var fn func([]byte) (map[string]interface{}, error)
	switch ext {
	case "js", "json":
		fallthrough
	case "yml", "yaml":
		fn = readYaml
		break
	default:
		return nil, fmt.Errorf("%s is not a valid yaml or json extension", ext)
	}
	fm, err := fn(buf)
	if err != nil {
		return nil, fmt.Errorf("cannot parse %s buf: %s", ext, err)
	}
	return fm, nil
}

func readJson(buf []byte) (map[string]interface{}, error) {
	return readUnmarshalableBuf(buf, json.Unmarshal)
}

func readYaml(buf []byte) (map[string]interface{}, error) {
	return readUnmarshalableBuf(buf, yaml.Unmarshal)
}

func readUnmarshalableBuf(buf []byte, unmarshal func([]byte, interface{}) error) (map[string]interface{}, error) {
	m := make(map[string]interface{})
	err := unmarshal(buf, &m)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshall: %s", err)
	}
	fm := flatten(m, dotJoiner)
	return fm, nil
}

var dotJoiner = func(a []string) string { return strings.Join(a, ".") }

var dotSlicer = func(s string) []string { return strings.Split(s, ".") }

func unflatten(flatmap map[string]interface{}, slicer func(string) []string) map[string]interface{} {
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

func flatten(unflatmap map[string]interface{}, joiner func([]string) string) map[string]interface{} {
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
