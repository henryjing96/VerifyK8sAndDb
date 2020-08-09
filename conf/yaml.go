package conf

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"
)

type K8sCluster struct {
	ConfigPath string   `yaml:"configPath"`
	FilteredNS []string `yaml:"filteredNS"`
}

type Mysql struct {
	Ip     string `yaml:"ip"`
	Port   string `yaml:"port"`
	User   string `yaml:"user"`
	Passwd string `yaml:"passwd"`
	Db     string `yaml:"db"`
}

type Alert struct {
	ChadId []string `yaml:"chatid"`
	Style  string   `yaml:"style"`
}

type YamlConfig struct {
	K8sCluster    K8sCluster `yaml:"k8sCluster"`
	Mysql         Mysql      `yaml:"mysql"`
	Alert         Alert      `yaml:"alert"`
	IsFilterNewVK bool       `yaml:"isFilterNewVK"`
}

var Cfg YamlConfig

func init() {
	configPath := "conf/server.yaml"
	LoadYamlConfig(configPath)
}

func LoadYamlConfig(configPath string) {
	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Load yaml file error: %v\n", err)
		os.Exit(1)
	}
	err = yaml.Unmarshal(yamlFile, &Cfg)
	if err != nil {
		fmt.Printf("Load yaml file error: %v\n", err)
		os.Exit(1)
	}
	prettyJson, err := json.MarshalIndent(Cfg, "", " ")
	if err != nil {
		fmt.Printf("json.Marshal fail %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("yaml config: %s\n", prettyJson)
}
