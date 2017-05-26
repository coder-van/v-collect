package statsd

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
	"strings"
)

type Config struct {
	IsLocal                bool   `toml:"is_local"`
	ReceiverAddr           string `toml:"receiver_addr"`
	ReceiverQueueSize      int    `toml:"receiver_queue_size"`
	GraphiteAddr           string `toml:"graphite_addr"`
	DataPointQueueSize     int    `toml:"datapoint_queue_size"`
	BackendFlushSeconds    int    `toml:"backend_flush_seconds"`
	BackendFlushSize       int    `toml:"backend_flush_size"`
	FlushSeconds 		   int    `toml:"flush_seconds"`
}

func NewConfig() *Config {
	return &Config{
		IsLocal:                true,
		ReceiverAddr:           ":2016",
		ReceiverQueueSize:      100000,
		GraphiteAddr:           ":2017",
		DataPointQueueSize:     100000,
		BackendFlushSeconds:    5,
		BackendFlushSize:       64,
		FlushSeconds:           5,
	}
}

func (c *Config) Check() {
	if !c.IsLocal && c.ReceiverAddr == ""  {
		panic("config is_local is false but receiver_addr empty")
	}
	if c.ReceiverQueueSize < 1024 {
		fmt.Println("warn config receiver_queue_size not can't smaller than 1024, set to 1M")
		c.ReceiverQueueSize = 1024*1024
	}
	if c.GraphiteAddr == "" || strings.Index(c.GraphiteAddr, ":") < 0{
		panic("config graphite_addr is invail")
	}
	if c.DataPointQueueSize < 1024 {
		fmt.Println("warn config datapoint_queue_size not in range, set to 1M")
		c.DataPointQueueSize = 1024*1024
	}
	if c.BackendFlushSize < 64 {
		fmt.Println("warn config backend_flush_size can't smaller than 64, set to 64")
		c.BackendFlushSize = 64
	}
	if c.BackendFlushSeconds < 1 {
		fmt.Println("warn config backend_flush_seconds can't smaller than 1, set to 5")
		c.BackendFlushSeconds = 5
	}
	if c.FlushSeconds < 1 {
		fmt.Println("warn config flush_seconds can't smaller than 1, set to 5")
		c.FlushSeconds = 5
	}
}

func (c *Config) LoadConfig(confPath string) (*Config, error) {

	if confPath, err := getDefaultConfigPath(confPath); err != nil {
		return nil, err
	} else {
		fmt.Printf("Loading config file: %s \n", confPath)
		if _, err := toml.DecodeFile(confPath, c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func getDefaultConfigPath(cp string) (string, error) {
	/*
	 Try to find a default config file at current or /etc dir
	*/

	confName := cp
	etcConfPath := "/etc/" + cp
	return getPath(etcConfPath, confName)
}

func getPath(paths ...string) (string, error) {
	fmt.Println("Search config file in paths:", paths)
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// if we got here, we didn't find a file in a default location
	return "", fmt.Errorf("Could not find path in %s", paths)
}
