//go:build !js

package main

const defaultFullscreen = false

const isWebBuild = false

func fullscreenSupported() bool {
	return true
}

func runningAsInstalledApp() bool {
	return false
}
