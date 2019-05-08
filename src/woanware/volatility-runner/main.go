package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	yaml "gopkg.in/yaml.v2"
)

// ##### Structures ####################################################################################################

type Config struct {
	VolatilityPath      string   `yaml:"volatility_path"`
	OutputPath          string   `yaml:"output_path"`
	UseSummaryDirectory bool     `yaml:"use_summary_dir"`
	Plugins             []Plugin `yaml:"plugins"`
	Data                []Data   `yaml:"data"`
}

type Data struct {
	RamPath string `yaml:"ram_path"`
	Profile string `yaml:"profile"`
}

type Plugin struct {
	Name      string `yaml:"name"`
	Summarise bool   `yaml:"summarise"`
}

// ##### Constants #####################################################################################################

// App Constants
const APP_NAME string = "volatility-runner (vr)"
const APP_VERSION string = "0.0.2"

// ##### Methods #######################################################################################################

//
func main() {

	fmt.Println(fmt.Sprintf("\n%s v%s - woanware\n", APP_NAME, APP_VERSION))

	var configPath string
	flag.StringVar(&configPath, "config", "", "Config file path")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s: %[1]s [OPTIONS]\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	flag.Parse()

	if len(configPath) == 0 {
		flag.PrintDefaults()
		return
	}

	c, err := loadConfig(configPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if validateConfig(c) == false {
		return
	}

	// If the user wants the summary files in a separate dir
	if c.UseSummaryDirectory == true {
		if _, err := os.Stat(filepath.Join(c.OutputPath, "summary")); os.IsNotExist(err) {
			os.Mkdir(filepath.Join(c.OutputPath, "summary"), 0775)
		}
	}

	for _, d := range c.Data {

		fileName := filepath.Base(d.RamPath)

		analysisFilePath := ""
		if c.UseSummaryDirectory == true {
			analysisFilePath = filepath.Join(c.OutputPath, "summary", fileName+".analysis.md")
		} else {
			analysisFilePath = filepath.Join(c.OutputPath, fileName+".analysis.md")
		}

		// Create analysis file
		var f *os.File
		f, err := os.OpenFile(analysisFilePath, os.O_CREATE|os.O_WRONLY, 0660)
		if err != nil {
			fmt.Printf("Error opening analysis file (%s): %v", fileName+".analysis.md", err)
			continue
		}
		defer f.Close()

		f.WriteString(fmt.Sprintln("# Summary"))
		f.WriteString(fmt.Sprintln(""))
		f.WriteString(fmt.Sprintf("- File: %s\n", d.RamPath))
		f.WriteString(fmt.Sprintf("- Profile: %s\n\n", d.Profile))

		for _, p := range c.Plugins {
			runCommand(f, c, d, p)
		}
	}
}

//
func runCommand(analysisFile *os.File, c *Config, data Data, plugin Plugin) {

	fileName := filepath.Base(data.RamPath)
	outputFilePath := filepath.Join(c.OutputPath, fileName+"."+plugin.Name+".txt")

	fmt.Println("Running plugin '" + plugin.Name + "' against RAM dump '" + fileName + "' @ " + time.Now().Format("2006-01-02T15:04:05"))

	if _, err := os.Stat(outputFilePath); os.IsNotExist(err) == false {
		fmt.Printf("Plugin already run: %s\n", outputFilePath)
		return
	}

	cmd := exec.Command(c.VolatilityPath, "-f", data.RamPath, "--profile", data.Profile, "--output-file", outputFilePath, plugin.Name)

	err := cmd.Start()
	if err != nil {
		fmt.Printf("Error running command: %v", err)
		return
	}
	cmd.Wait()

	if plugin.Summarise == false {
		return
	}

	if _, err := os.Stat(outputFilePath); err == nil {

		dataFile, err := os.OpenFile(outputFilePath, os.O_RDONLY, 0644)
		if err != nil {
			fmt.Printf("Error opening plugin data file (%s): %v\n", outputFilePath, err)
			return
		}

		data, err := ioutil.ReadAll(dataFile)
		if err != nil {
			fmt.Printf("Error reading plugin data file (%s): %v\n", outputFilePath, err)
			return
		}

		analysisFile.WriteString(fmt.Sprintf("## %s\n\n```\n", plugin.Name))
		analysisFile.WriteString(string(data))
		analysisFile.WriteString(fmt.Sprintln("\n```\n"))
	}
}

// loadConfig validates the config path, and unmarshals the YAML into a struct
func loadConfig(path string) (*Config, error) {

	if _, err := os.Stat(path); err != nil {
		return nil, errors.New("config file does not exist")
	}

	input, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open config file: %v", err)
	}
	defer input.Close()

	temp, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file: %v", err)
	}

	var c *Config
	err = yaml.Unmarshal([]byte(temp), &c)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshall config file: %v", err)
	}

	return c, nil
}

//
func validateConfig(c *Config) bool {

	ret := true
	if _, err := os.Stat(c.VolatilityPath); os.IsNotExist(err) {
		fmt.Println("Volatility path does not exist")
		ret = false
	}

	if len(c.OutputPath) == 0 {
		fmt.Println("Output path not set")
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
