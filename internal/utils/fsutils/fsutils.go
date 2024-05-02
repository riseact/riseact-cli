package fsutils

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func IsDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}

	return fileInfo.IsDir()
}

func GetMimeType(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return "", err
	}

	mimeType := http.DetectContentType(buffer)
	return mimeType, nil
}

func ValidateEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue

		}

		if !strings.Contains(line, "=") {
			return fmt.Errorf("invalid key-value pair on line %d", lineNum)
		}

		for _, char := range line {
			if char == ';' || char == '#' {
				return fmt.Errorf("invalid character on line %d", lineNum)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil

}
