package config

import (
	"encoding/json"

	"cuelang.org/go/cue"
)

type Config struct {
	APIVersion string
	Kind       string
	Namespace  string
	Events     []string
}

// TODO: validate config

func (c *Config) Parse(vConf cue.Value) error {
	js, err := vConf.MarshalJSON()
	if err != nil {
		return err
	}
	err = json.Unmarshal(js, c)
	if err != nil {
		return err
	}

	return nil
}
