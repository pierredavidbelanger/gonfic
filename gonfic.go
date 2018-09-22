package gonfic

import (
	"encoding/json"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/wolfeidau/unflatten"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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
	return unflatten.Unflatten(c.ToFlatMap(), unflatten.SplitByDot)
}

// Unmarshal the keys and values as an hierarchical map
// and stores the result in the value pointed to by v.
func (c *Config) Unmarshal(v interface{}) error {
	return mapstructure.Decode(c.ToHierarchicalMap(), v)
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
		fn = readJson
		break
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
	fm := unflatten.Flatten(m, unflatten.JoinWithDot)
	return fm, nil
}
