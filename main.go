package main

import (
	"encoding/json"
	"log"
	"os"
)

func main() {
	configFile := "config.json"

	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Failed to read configuration file: %s", configFile)
	}

	var repositories []Repository
	err = json.Unmarshal(data, &repositories)
	if err != nil {
		log.Fatalf("Failed to parse configuration file: %s. Error: %v", configFile, err)
	}

	for _, repo := range repositories {
		ctx := RepositoryContext{Repo: repo}
		ctx.ProcessRepository()
	}
}
