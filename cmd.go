package main

import (
	"context"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v2"
)

type setting struct {
	Path     string   `yaml:"path"`
	Commands []string `yaml:"commands"`
}

func parseConfig() (*setting, error) {
	yamlFile, err := ioutil.ReadFile("config.yml")
	if err != nil {
		return nil, err
	}

	var s []setting
	err = yaml.Unmarshal(yamlFile, &s)
	if err != nil {
		return nil, err
	}

	return &s[0], nil
}

func (c *setting) execCommands(ctx context.Context) (string, error) {
	for _, cmd := range c.Commands {
		cmdArgs := strings.Fields(cmd)
		out, err := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...).Output()
		if err != nil {
			log.Fatal("Error executing command:", err)
		}
		return string(out), nil
	}

	return "", nil
}

func (c setting) path() string {
	return c.Path
}
