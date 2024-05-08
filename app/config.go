package app

import (
	"fmt"
	"log"
	"os"

	"github.com/SUSE/telemetry/pkg/config"
	"gopkg.in/yaml.v3"
)

// Default server config path
const DEFAULT_CONFIG string = "/etc/susetelemetry/server.cfg"

// Datastore config for staging the data
//type DataStoresConfig struct {
//	ItemDS   string `yaml:"items"`
//	BundleDS string `yaml:"bundles"`
//	ReportDS string `yaml:"reports"`
//}

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

type Config struct {
	cfgPath    string
	API        APIConfig               `yaml:"api"`
	DataStores config.DataStoresConfig `yaml:"datastores"`
	DataBases  struct {
		Telemetry DBConfig `yaml:"telemetry"`
		//add other databases here
	} `yaml:"dbs"`
}

func NewConfig(cfgFile string) *Config {
	cfg := &Config{cfgPath: cfgFile}

	return cfg
}

func (cfg *Config) Path() string {
	return cfg.cfgPath
}

func (cfg *Config) Load() error {
	log.Printf("cfgPath: %q", cfg.cfgPath)
	_, err := os.Stat(cfg.cfgPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("config file '%s' doesn't exist: %s", cfg.cfgPath, err)
	}

	contents, err := os.ReadFile(cfg.cfgPath)
	if err != nil {
		return fmt.Errorf("failed to read contents of config file '%s': %s", cfg.cfgPath, err)
	}

	log.Printf("Contents: %q", contents)
	err = yaml.Unmarshal(contents, cfg)
	if err != nil {
		return fmt.Errorf("failed to parse contents of config file '%s': %s", cfg.cfgPath, err)
	}

	return nil
}
