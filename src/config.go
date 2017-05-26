package main

/* load config file 加载配置文件 */

import (
	"fmt"
	"os"
	"github.com/BurntSushi/toml"
	statsd "github.com/coder-van/v-stats"
	"github.com/coder-van/v-collect/src/collector"
)

func NewConfig() *Config {
	return &Config{
		AgentDefaultName: "v-collect-agent",
		StatsdConfig:     *statsd.NewConfig(),
		
	}
}


type Config struct {
	Debug            bool          `toml:"debug"`
	AgentDefaultName string        `toml:"agent_name"`
	CollectSeconds   int           `toml:"collect_seconds"`
	HostConfig       HostConfig    `toml:"host"`
	LoggingConfig    LoggingConfig `toml:"logging"`
	StatsdConfig     statsd.Config   `toml:"statsd"`
	CollectorConf    CollectorConfig `toml:"collector"`
}

type HostConfig struct {
	HostGroup  string `toml:"host_group"`
	LicenseKey string `toml:"license_key"`
	HostSign   string `toml:"host_sign"`
}

type LoggingConfig struct {
	LogLevel string `toml:"log_level"`
	LogDir   string `toml:"log_dir"`
}

type CollectorConfig struct {
	ProcConf  collector.ProcConfig  `toml:"proc"`
	NginxConf collector.NginxConfig `toml:"nginx"`
}

func LoadConfig(confPath string) (*Config, error) {
	c := NewConfig()
	var cp string = confPath
	var err error
	if confPath == "" {
		cp, err = getDefaultConfigPath()
		if err != nil {
			return nil, err
		}
	}
	fmt.Printf("--- Loading config file: %s ---\n", cp)
	if _, err := toml.DecodeFile(cp, c); err != nil {
		return nil, err
	}
	SetDefault(c)
	return c, nil
}

func SetDefault(c *Config) {
	if c.HostConfig.HostGroup == "" {
		c.HostConfig.HostGroup = "group"
	}
	if c.HostConfig.HostSign == "" {
		c.HostConfig.HostSign = "sign"
	}

	if c.LoggingConfig.LogDir == "" {
		c.LoggingConfig.LogDir = "/var/log/v-collect"
		fmt.Println("log dir not set, use /var/log/v-collect")
	}
	if _, err := os.Stat(c.LoggingConfig.LogDir); err != nil {
		panic("Log dir not exist, " + c.LoggingConfig.LogDir)
	}
	if c.CollectSeconds == 0 {
		c.CollectSeconds = 1
	}
}

func getDefaultConfigPath() (string, error) {
	/*
	 Try to find a default config file at current or /etc dir
	*/
	etcConfPath := "/etc/v-collect/default.ini"
	return getPath(etcConfPath, "./conf/default.ini")
}

func getPath(paths ...string) (string, error) {
	fmt.Println("* Search config file in paths:", paths)
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// if we got here, we didn't find a file in a default location
	panic("Could not find path ")
}
