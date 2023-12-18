package main

import (
	"fmt"
	"os"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Config represents the structure of the configuration file with all the 
// different attributes that can be configured.
type Config struct {
	RemoteWriteURLs    string           `yaml:"remoteWriteURLs"`    // Comma-separated list of remote write endpoints
	Debug              bool             `yaml:"debug"`               // If true, print metrics to stdout
	Workers            int              `yaml:"workers"`             // Number of worker goroutines
	BatchSize          int              `yaml:"batchSize"`           // Number of TimeSeries to batch before sending
	ReleaseAfterSeconds int             `yaml:"releaseAfterSeconds"` // Number of seconds to wait before releasing a batch
	JsonMinCardinality int              `yaml:"jsonMinCardinality"`  // Minimum cardinality to include in the output JSON
  StatsIntervalSeconds int `yaml:"stats_interval_seconds"`// StatsIntervalSeconds field of type int
	CardinalityLimit   CardinalityLimit `yaml:"cardinalityLimit"`    // Config parameters for cardinality limit
}

// CardinalityLimit represents the parameters for cardinality limitation.
type CardinalityLimit struct {
  Enable   bool   `yaml:"enable"`   // Enable or Disable the Cardinality plugin
	Capacity int    `yaml:"capacity"` // Number of unique values to store per tag
	Limit    int    `yaml:"limit"`    // Cardinality threshold to trigger limit action
	Mode     string `yaml:"mode"`     // Mode of cardinality limitation: drop_metric, drop_tag
}

// readConfig reads a config file and unmarshals it into a Config struct
func (c *Config) readConfig(configFile string) *Config {
	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	return c
}
