package main

import (
	"fmt"
	"testing"
)

func TestProcessDemoFile(t *testing.T) {
	fmt.Println("initiating test")
	destFolder := "/home/marcel/projetos/data_ssd/profiling"
	dirExists, _ := exists(destFolder)
	if dirExists {
		RemoveContents(destFolder)
	}
	ProcessDemoFile("/home/marcel/projetos/data/csgo_analyze/replays/mibr-vs-chaos-inferno.dem", 0,
		destFolder, 128)
}
