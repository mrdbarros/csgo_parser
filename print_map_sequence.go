package main

import (
	"fmt"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"

	r3 "github.com/golang/geo/r3"
	dem "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
	metadata "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/metadata"
	utils "github.com/mrdbarros/csgo_analyze/utils"
)

var currentState = ""
var gameReset = false
var gameStarted = false
var discretizeFactor = 20.0
var roundStartTime int
var lastUpdate = 0
var lastTimeEvent = 0
var tickRate = 0
var posUpdateInterval = 2

type playerMapping struct {
	playerSeqID int
	position    r3.Vector
}

var trMap = make(map[int]playerMapping)
var ctMap = make(map[int]playerMapping)

// exists returns whether the given file or directory exists
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func checkError(err error) {
	if err != nil {
		print("error!")
		panic(err)
	}
}

func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func processDemoFile(demPath string, fileID int, destDir string) {
	f, err := os.Open(demPath)
	checkError(err)
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Erro no processamento do arquivo!", r)
		}
	}()
	p := dem.NewParser(f)
	defer f.Close()
	header, err := p.ParseHeader()
	checkError(err)
	fmt.Println("Map:", header.MapName)
	mapName := header.MapName
	dirName := destDir + "/" + header.MapName + "/" + strconv.Itoa(fileID)
	dirExists, _ := exists(dirName)
	if !dirExists {
		err = os.MkdirAll(dirName, 0700)
		checkError(err)
	}

	newFile := dirName + "/" + header.MapName + "_" + strconv.Itoa(fileID) + ".txt"
	fileWrite, err := os.Create(newFile)
	checkError(err)

	defer fileWrite.Close()
	gameReset := false
	gameStarted := false
	winTeamCurrentRound := "t"
	roundDir := dirName
	fullMap := utils.AnnotatedMap{IconsList: nil, SourceMap: header.MapName}
	mapMetadata := metadata.MapNameToMap[header.MapName]
	imageIndex := 0
	p.RegisterEventHandler(func(e events.FrameDone) {
		gs := p.GameState()
		if !(gs == nil) && roundDir != dirName {
			processFrameEnd(gs, p, &fullMap, mapMetadata, &imageIndex, roundDir)
		}

	})

	p.RegisterEventHandler(func(e events.RoundStart) {
		gs := p.GameState()
		if !(gs == nil) {
			if gs.TeamCounterTerrorists().Score() == 0 && gs.TeamTerrorists().Score() == 0 && !gameStarted {
				gameReset = true
				RemoveContents(dirName)
			}
			if gs.TeamCounterTerrorists().Score()+gs.TeamTerrorists().Score() > 10 && gameReset {
				gameStarted = true
			}
		}
		newScore := "ct_" + strconv.Itoa(gs.TeamCounterTerrorists().Score()) +
			"_t_" + strconv.Itoa(gs.TeamTerrorists().Score())
		roundDir = dirName + "/" + newScore
		dirExists, _ := exists(roundDir)
		imageIndex = 0
		if !dirExists {
			err = os.MkdirAll(roundDir, 0700)
			checkError(err)
		} else {
			RemoveContents(roundDir)
		}
		roundStartTime = getCurrentTime(p)
	})

	p.RegisterEventHandler(func(e events.RoundEnd) {

		win_team := e.Winner
		if win_team == 2 {
			winTeamCurrentRound = "t"
		} else if win_team == 3 {
			winTeamCurrentRound = "ct"
		} else {
			winTeamCurrentRound = "invalid"
		}
	})

	p.RegisterEventHandler(func(e events.RoundEndOfficial) {
		//rename round folder with winner team
		if roundDir != dirName {
			fileWrite, err := os.Create(roundDir + "/winner.txt")
			checkError(err)
			defer fileWrite.Close()
			_, err = fileWrite.WriteString(winTeamCurrentRound)
			checkError(err)
		}

	})
	err = p.ParseToEnd()
	checkError(err)
	if currentState[0:3] != "de_" {
		currentState = mapName + " " + currentState
	}
	_, err = fileWrite.WriteString(currentState)
	checkError(err)
	// Parse to end
}

func main() {
	demPath := os.Args[1]
	destDir := os.Args[2]

	tickRate, _ = strconv.Atoi(os.Args[3])
	fileID := 0
	files, err := ioutil.ReadDir(demPath)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		processDemoFile(demPath+"/"+f.Name(), fileID, destDir)
		fileID++
	}

}

func processPlayerHP(gs dem.GameState, fullMap *utils.AnnotatedMap) {
	tr := gs.TeamTerrorists()
	ct := gs.TeamCounterTerrorists()
	(*fullMap).IconsList = nil
	//add t icons
	for _, t := range tr.Members() {

	}
	//add ct icons
	for _, ct := range ct.Members() {

	}
}

func processPlayerInformation(gs dem.GameState, fullMap *utils.AnnotatedMap,
	mapMetadata metadata.Map) {

	processPlayerPositions(gs, fullMap, mapMetadata)
}

func processFrameEnd(gs dem.GameState, p dem.Parser,
	fullMap *utils.AnnotatedMap, mapMetadata metadata.Map, imageIndex *int,
	roundDir string) {
	currentTime := getCurrentTime(p)
	currentRoundTime := getRoundTime(p)
	if currentRoundTime%posUpdateInterval == 0 && currentTime != lastUpdate {
		lastUpdate = currentTime

		processPlayerInformation(gs, fullMap, mapMetadata)
		generateMap(fullMap, roundDir, imageIndex)
	}
}

func getRoundTime(p dem.Parser) int {
	return int(getCurrentTime(p) - roundStartTime)
}

func getCurrentTime(p dem.Parser) int {
	return p.CurrentFrame() / tickRate
}

func generateMap(fullMap *utils.AnnotatedMap, roundDir string, imageIndex *int) {
	img := utils.DrawMap(*fullMap)
	third, err := os.Create(roundDir + "/output_map" +
		strconv.Itoa(*imageIndex) + ".jpg")
	if err != nil {
		log.Fatalf("failed to create: %s", err)
	}
	jpeg.Encode(third, img, &jpeg.Options{jpeg.DefaultQuality})
	*imageIndex++
	defer third.Close()
}

func processPlayerPositions(gs dem.GameState, fullMap *utils.AnnotatedMap,
	mapMetadata metadata.Map) {
	tr := gs.TeamTerrorists()
	ct := gs.TeamCounterTerrorists()
	(*fullMap).IconsList = nil
	//add t icons
	for _, t := range tr.Members() {

		x, y := mapMetadata.TranslateScale(t.Position().X, t.Position().Y)
		newIcon := utils.Icon{X: x - 10, Y: y - 10, IconName: "terrorist_1"}
		(*fullMap).IconsList = append((*fullMap).IconsList, newIcon)
	}
	//add ct icons
	for _, ct := range ct.Members() {

		x, y := mapMetadata.TranslateScale(ct.Position().X, ct.Position().Y)
		newIcon := utils.Icon{X: x - 10, Y: y - 10, IconName: "ct_1"}
		(*fullMap).IconsList = append((*fullMap).IconsList, newIcon)
	}

}
