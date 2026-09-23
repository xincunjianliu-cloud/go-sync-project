//go:build !js

package main

func decodeMP3ToPCM(data []byte) ([]byte, error) {
	return decodeMP3ToPCMGo(data)
}
