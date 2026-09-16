//go:build !js

package main

import "os"

func readRuntimeFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func writeRuntimeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func runtimeFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
