package api

import (
	"fmt"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

const (
	minKeyLen = 60
)

type Config struct {
	TTL        time.Duration     `yaml:"ttl"` // Time to live.
	DomainKeys map[string]string `yaml:"keys"`
}

func parseConfig(path string) (c *Config, err error) {
	// Parse the spec.
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	c = &Config{}
	err = yaml.UnmarshalStrict(data, c)
	if err != nil {
		return
	}

	if c.TTL.Seconds() <= 0 {
		err = fmt.Errorf("invalid ttl: %v", c.TTL)
		return
	}

	var d string
	for k, v := range c.DomainKeys {
		d, err = toValidDomain(k)
		if err != nil {
			return
		} else if k != d {
			err = fmt.Errorf("invalid domain: %s != %s", k, d)
			return
		}

		if len(v) < minKeyLen {
			err = fmt.Errorf("key must have %v characters", minKeyLen)
			return
		}
	}

	return
}
