package main

import (
	yaml "gopkg.in/yaml.v2"
	"os"
	"time"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"os/exec"
)

// ##### Structures ####################################################################################################

type Config struct {
	VolatilityPath string   `yaml:"volatility_path"`
	OutputPath     string   `yaml:"output_path"`
	Plugins        []string `yaml:"plugins"`
	Data           []Data   `yaml:"data"`
}

type Data struct {
	RamPath string `yaml:"ram_path"`
	Profile string `yaml:"profile"`
}

// ##### Constants #####################################################################################################

// App Constants
const APP_NAME string = "volatility-runner (vr)"
const APP_VERSION string = "0.0.1"

// ##### Methods #######################################################################################################

//
func main() {

	fmt.Println(fmt.Sprintf("\n%s v%s - woanware\n", APP_NAME, APP_VERSION))

	c, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v", err)
		return
	}

	if validateConfig(c) == false {
		return
	}

	for _, p := range c.Plugins {
		for _, d := range c.Data {		
			runCommand(c, d, p)
		}
	}
}

//
func runCommand(c *Config, data Data, plugin string) {

	fileName := filepath.Base(data.RamPath)
	fmt.Println("Running plugin '" + plugin + "' against RAM dump '" + fileName + "' @ " +  time.Now().Format(time.RFC3339))
	cmd := exec.Command(c.VolatilityPath, "-f", data.RamPath, "--profile", data.Profile, "--output-file", filepath.Join(c.OutputPath, fileName + "." + plugin + ".txt"), plugin)

    err := cmd.Start()
    if err != nil {
		fmt.Printf("Error running command: %v", err)
	}
    cmd.Wait()
}

//
func loadConfig() (*Config, error) {

	input, err := os.Open("volatility-runner.config")
	if err != nil {
		return nil, fmt.Errorf("Error opening hunt file: %v\n", err)
	}
	defer input.Close()

	temp, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("Error reading hunt file: %v\n", err)
	}

	var c *Config
	err = yaml.Unmarshal([]byte(temp), &c)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling the hunt file: %v\n", err)
	}

	return c, nil
}

// 
func validateConfig(c *Config) bool {

	ret := true
	if _, err := os.Stat(c.VolatilityPath); err != nil {
		fmt.Println("Volatility path does not exist")
		ret = false
	}

	for _, d := range c.Data {
		if _, err := os.Stat(d.RamPath); err != nil {
			fmt.Printf("RAM path does not exist: %s\n", d.RamPath)
			ret = false
		}
	}

	return ret
}
