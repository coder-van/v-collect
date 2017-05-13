package main
/* load config file 加载配置文件 */

import (
	"os"
	"fmt"
	statsd "github.com/coder-van/v-stats"
	"github.com/BurntSushi/toml"
)

func NewConfig() *Config {
	return &Config{AgentDefaultName: "v-collect-agent"}
}

type Config struct {
	Debug            bool          `toml:"debug"`
	AgentDefaultName string        `toml:"agent_name"`
	CollectSeconds   int           `toml:"collect_seconds"`
	HostConfig       HostConfig    `toml:"host"`
	LoggingConfig    LoggingConfig `toml:"logging"`
	StatsdConfig     StatsdConfig  `toml:"statsd"`
}

type HostConfig struct {
	HostGroup 		string  `toml:"host_group"`
	LicenseKey      string  `toml:"license_key"`
	HostSign        string  `toml:"host_sign"`
}

type LoggingConfig struct {
	LogLevel string `toml:"log_level"`
	LogDir  string `toml:"log_dir"`
}

type StatsdConfig  *statsd.Config

func LoadConfig(confPath string) (*Config, error) {
	c := NewConfig()
	if confPath == "" {
		if confPath, err := getDefaultConfigPath(c.AgentDefaultName); err != nil {
			return nil, err
		}else{
			fmt.Printf("Loading config file: %s \n", confPath)
			if _, err := toml.DecodeFile(confPath, c); err != nil {
				return nil, err
			}
		}
	}
	SetDefault(c)
	return c, nil
}

func SetDefault(c *Config)  {
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
		panic("Log dir not exist, "+c.LoggingConfig.LogDir)
	}
	if c.CollectSeconds == 0 {
		c.CollectSeconds = 1
	}
}

func getDefaultConfigPath(agentDefaultName string) (string, error) {
	/*
	 Try to find a default config file at current or /etc dir
	 */
	defaultPath := "./bin/agent.conf"
	confName := agentDefaultName+".conf"
	etcConfPath := "/etc/"+agentDefaultName+"/"+confName
	return getPath(etcConfPath, confName, defaultPath)
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