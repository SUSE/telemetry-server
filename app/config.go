package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

// Default server config path
const DEFAULT_CONFIG string = "/etc/susetelemetry/server.cfg"

// API server config
type APIConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type PQLConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	SSLMode  string `yaml:"sslmode"`
	Cert     string `yaml:"cert"`
}

type DBConfig struct {
	Driver string `yaml:"driver"`
	Params string `yaml:"params"`
}

func (d *DBConfig) Valid() error {
	return nil
}

type LogConfig struct {
	Level    string `yaml:"level" json:"level"`
	Location string `yaml:"location" json:"location"`
	Style    string `yaml:"style" json:"style"`
}

func (lc *LogConfig) String() string {
	str, _ := json.Marshal(lc)
	return string(str)
}

type Config struct {
	cfgPath string
	API     APIConfig `yaml:"api"`
	//DataStores config.DBConfig `yaml:"datastores"`
	DataBases struct {
		Telemetry   DBConfig `yaml:"telemetry"`
		Operational DBConfig `yaml:"operational"`
		Staging     DBConfig `yaml:"staging"`
		//add other databases here
	} `yaml:"dbs"`
	Logging LogConfig `yaml:"logging"`
}

func NewConfig(cfgFile string) *Config {
	cfg := &Config{cfgPath: cfgFile}

	return cfg
}

func (cfg *Config) Path() string {
	return cfg.cfgPath
}

func (cfg *Config) Load() error {
	slog.Debug("Loading config", slog.String("path", cfg.cfgPath))
	_, err := os.Stat(cfg.cfgPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("config file '%s' doesn't exist: %s", cfg.cfgPath, err)
	}

	contents, err := os.ReadFile(cfg.cfgPath)
	if err != nil {
		return fmt.Errorf("failed to read contents of config file '%s': %s", cfg.cfgPath, err)
	}

	slog.Debug("Loaded config", slog.String("contents", string(contents)))
	err = yaml.Unmarshal(contents, cfg)
	if err != nil {
		return fmt.Errorf("failed to parse contents of config file '%s': %s", cfg.cfgPath, err)
	}

	return nil
}
