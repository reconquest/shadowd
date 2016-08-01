package main

import "github.com/kovetskiy/ko"

type config struct {
	Backend struct {
		Use  string `toml:"use" required:"true"`
		Path string `toml:"path" required:"true"`
	} `toml:"backend" required:"true"`
}

func getConfig(path string) (*config, error) {
	config := &config{}
	err := ko.Load(path, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
