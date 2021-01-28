package main

import (
	"fmt"
	"testing"

	"github.com/mrdbarros/csgo_analyze/utils"
)

func TestProcessDemoFile(t *testing.T) {
	fmt.Println("initiating test")
	destFolder := "/home/marcel/projetos/data/csgo_analyze/processed/gc/"
	dirExists, _ := utils.Exists(destFolder)
	if dirExists {
		utils.RemoveContents(destFolder)
	}
	ProcessDemoFile("/home/marcel/projetos/data/csgo_analyze/replays/0/2021-01-07__1843__1__10374236__de_nuke__timedegolion__vs__c4base.dem", 0,
		destFolder, 128)
}
