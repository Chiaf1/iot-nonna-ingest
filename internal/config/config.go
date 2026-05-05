package config

import (
	"fmt"
	"os"
	"time"

	"github.com/chiaf1/iot-nonna-ingest/internal/utils"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Broker             string        `yaml:"broker"`
	ClientID           string        `yaml:"clientId"`
	QoS                uint8         `yaml:"qos"`
	ConnectionInterval time.Duration `yaml:"connectionInterval"`
	MaxRetry           int           `yaml:"maxRetry"`
	MaxDelay           time.Duration `yaml:"maxDelay"`
	Topics             []string      `yaml:"topics"`
}

// Load loads the values frrom the file "path" to the struct c, if the file is not present:
// the default values are loaded and the file is created.
func (c *Config) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.SetDefault()
			c.Save(path)
			return nil
		}
		return fmt.Errorf("Error while reading the config file: %w", err)
	}
	err = yaml.Unmarshal(data, c)
	if err != nil {
		return fmt.Errorf("Error during parsing of YAML config file: %w", err)
	}
	return nil
}

// SetDefault sets the config default values
func (c *Config) SetDefault() {
	c.Broker = "tcp://localhost:1883"
	c.ClientID = "go-mqtt-client"
	c.QoS = 1
	c.ConnectionInterval = 3 * time.Second
	c.MaxRetry = 0
	c.MaxDelay = 60 * time.Second
	c.Topics = []string{
		"topic/test",
	}
}

// Save saves the configs to the "path" using the WriteFileAtomic function
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("Error while parsing to YAML: %w", err)
	}
	return utils.WriteFileAtomic(path, data, 0644)
}

// Data validation after loading
func (c *Config) Validate() error {
	if c.Broker == "" {
		return fmt.Errorf("broker cannot be empty")
	}
	if len(c.Topics) == 0 {
		return fmt.Errorf("no topics defined")
	}
	return nil
}
