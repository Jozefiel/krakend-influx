package influxdb2

import (
	"errors"
	"time"

	"github.com/luraproject/lura/v2/config"
)

type influx2Config struct {
	address    	string
	token   	string
	org   		string
	bucket   	string
	batchSize 	int
	ttl        	time.Duration
}

func configGetter(extraConfig config.ExtraConfig) interface{} {
	value, ok := extraConfig[Namespace]
	if !ok {
		return nil
	}

	castedConfig, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}

	cfg := influx2Config{}

	if value, ok := castedConfig["address"]; ok {
		cfg.address = value.(string)
	}

	if value, ok := castedConfig["token"]; ok {
		cfg.token = value.(string)
	}

	if value, ok := castedConfig["org"]; ok {
		cfg.org = value.(string)
	}

	if value, ok := castedConfig["bucket"]; ok {
		cfg.bucket = value.(string)
	} else {
		cfg.bucket = "krakend"
	}

	if value, ok := castedConfig["batch_size"]; ok {
		if s, ok := value.(int); ok {
			cfg.batchSize = s
		}
	}

	if value, ok := castedConfig["ttl"]; ok {
		s, ok := value.(string)

		if !ok {
			return nil
		}
		var err error
		cfg.ttl, err = time.ParseDuration(s)

		if err != nil {
			return nil
		}
	}

	return cfg
}

var ErrNoConfig = errors.New("influxdb2: unable to load custom config")
