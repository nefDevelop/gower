package main

import (
	"gower/cmd"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	cmd.SetVersionInfo(Version, BuildTime)
	cmd.Execute()
}
