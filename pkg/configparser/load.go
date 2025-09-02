package configparser

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

var ErrNoFilePath = errors.New("no file path provided")

// LoadYamlFile reads a YAML file and loads variables into the environment
func LoadYamlFile(filepath string) error {
	if filepath == "" {
		return ErrNoFilePath
	}

	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("could not open YAML file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	prefixStack := []string{}
	previousIndent := 0

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// Calculate indentation (count leading spaces)
		indent := 0
		for _, ch := range line {
			if ch != ' ' {
				break
			}
			indent++
		}

		// Update prefix stack based on indentation changes
		if indent < previousIndent {
			// Pop elements from stack until we reach the correct level
			levelsToPop := (previousIndent - indent) / 2
			for i := 0; i < levelsToPop && len(prefixStack) > 0; i++ {
				prefixStack = prefixStack[:len(prefixStack)-1]
			}
		}
		previousIndent = indent

		// Remove indentation from line
		content := strings.TrimSpace(line)

		// Check if it's a section (ends with colon but not a key-value pair)
		if strings.HasSuffix(content, ":") && !strings.Contains(content, ": ") {
			// This is a new section
			sectionName := strings.TrimSuffix(content, ":")
			prefixStack = append(prefixStack, sectionName)
			continue
		}

		// Parse key-value pair
		parts := strings.SplitN(content, ":", 2)
		if len(parts) != 2 {
			continue // Skip malformed lines
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Handle empty values (just "key:")
		if value == "" {
			continue // Skip empty values, they don't represent environment variables
		}

		// Remove quotes if present
		value = strings.Trim(value, `"'`)

		// Handle environment variable substitution syntax: ${VAR:-default}
		if strings.HasPrefix(value, "${") && strings.Contains(value, ":-") && strings.HasSuffix(value, "}") {
			// Extract the variable name and default value
			inner := value[2 : len(value)-1] // Remove ${ and }
			subParts := strings.SplitN(inner, ":-", 2)
			if len(subParts) == 2 {
				envVarName := strings.TrimSpace(subParts[0])
				defaultValue := strings.TrimSpace(subParts[1])

				// Check if environment variable is already set
				if envValue := os.Getenv(envVarName); envValue != "" {
					value = envValue
				} else {
					value = defaultValue
				}
			}
		}

		// Build the full env var name with prefixes
		fullKey := strings.ToUpper(key)
		if len(prefixStack) > 0 {
			fullKey = strings.ToUpper(strings.Join(append(prefixStack, key), "_"))
		}

		// Set the environment variable only if it's not already set
		if os.Getenv(fullKey) == "" {
			if err := os.Setenv(fullKey, value); err != nil {
				return fmt.Errorf("could not set env var %s: %w", fullKey, err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading YAML file: %w", err)
	}

	return nil
}
