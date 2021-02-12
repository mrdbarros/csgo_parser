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
	ProcessDemoFile("/home/marcel/projetos/data/csgo_analyze/replays/0/2021-01-27__2053__1__10613458__de_inferno__timewess__vs__c4base.dem", 0,
		destFolder, 32)
}
