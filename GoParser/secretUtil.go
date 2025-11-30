package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type SetupConfiguration struct {
	Token   string
	CSVPath string
}

func SetupEnvironment() (SetupConfiguration, error) {
    config := SetupConfiguration{}
    
    secretsPath := os.Getenv("GOPARSER_SECRETS_PATH")
    if secretsPath != "" {
        if err := loadEnvFile(secretsPath); err != nil {
            return SetupConfiguration{}, fmt.Errorf("failed to load secrets from %s: %w", secretsPath, err)
        }
    }
    
    token := os.Getenv("GITHUB_TOKEN")
    if token == "" {
        return SetupConfiguration{}, fmt.Errorf("GITHUB_TOKEN not found - set GOPARSER_SECRETS_PATH or use VSCode launch.json")
    }
    
    csvPath := os.Getenv("CSV_PATH")
    if csvPath == "" {
        csvPath = "../input/alleSourcegraph.csv"
    }

    config.Token = token
    config.CSVPath = csvPath
    return config, nil
}

func loadEnvFile(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("could not open env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Split by first '=' only
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// Remove quotes if present
		value = strings.Trim(value, "\"'")
		
		os.Setenv(key, value)
	}
	
	return scanner.Err()
}