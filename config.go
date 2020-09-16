package main

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// YamlConfig ...
type YamlConfig struct {
	Setting AWSAccount `yaml:"Setting"`
}

// AWSAccount ...
type AWSAccount struct {
	Source      awsAuth `yaml:"Source"`
	Destination awsAuth `yaml:"Destination"`
	DryRun      bool    `yaml:"DryRun"`
}

type awsAuth struct {
	AccessKey    string `yaml:"AccessKey"`
	SecretKey    string `yaml:"SecretKey"`
	Region       string `yaml:"Region"`
	VIPCID       string `yaml:"VPCID"`
	HostedZoneID string `yaml:"HostedZoneID"`
}

// GetConfig ...
func GetConfig(configPath string) *YamlConfig {
	var yc YamlConfig
	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Println(err)
	}

	err = yaml.Unmarshal(yamlFile, &yc)
	if err != nil {
		fmt.Println("Unmarshal:", err)
	}

	return &yc
}
