package config

import (
	"fmt"
	"os"
	"time"

	"github.com/chiaf1/iot-nonna-ingest/internal/utils"
	"gopkg.in/yaml.v3"
)

type Config struct {
	MQTT    MqttConfig    `yaml:"MQTT"`
	DB      DbConfig      `yaml:"DB"`
	Workers WorkersConfig `yaml:"Workers"`
}

type MqttConfig struct {
	Broker             string        `yaml:"broker"`
	ClientID           string        `yaml:"clientId"`
	QoS                uint8         `yaml:"qos"`
	ConnectionInterval time.Duration `yaml:"connectionInterval"`
	MaxRetry           int           `yaml:"maxRetry"`
	MaxDelay           time.Duration `yaml:"maxDelay"`
}

type DbConfig struct {
	DbURL                   string        `yaml:"DbURL"`
	Query_timeout_read      time.Duration `yaml:"query_timeout_read"`
	Query_timeout_write     time.Duration `yaml:"query_timeout_write"`
	TopicRefreshingInterval time.Duration `yaml:"topicRefreshingInterval"`
}

type WorkersConfig struct {
	Number int `yaml:"number"`
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
	c.MQTT.Broker = "tcp://localhost:1883"
	c.MQTT.ClientID = "go-mqtt-client"
	c.MQTT.QoS = 1
	c.MQTT.ConnectionInterval = 3 * time.Second
	c.MQTT.MaxRetry = 0
	c.MQTT.MaxDelay = 60 * time.Second

	c.DB.DbURL = "postgres://user:password@localhost:5432/mydb?sslmode=disable"
	c.DB.Query_timeout_read = 5 * time.Second
	c.DB.Query_timeout_write = 2 * time.Second
	c.DB.TopicRefreshingInterval = 60 * time.Second

	c.Workers.Number = 5
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
	if c.MQTT.Broker == "" {
		return fmt.Errorf("[config][mqtt] broker cannot be empty")
	}
	if c.DB.DbURL == "" {
		return fmt.Errorf("[config][db] connection url cannot be empty")
	}
	if c.Workers.Number == 0 {
		return fmt.Errorf("[config][workers] Workers number cannot be 0")
	}
	return nil
}
