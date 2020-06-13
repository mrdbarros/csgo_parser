package main

import (
	"encoding/csv"
	"fmt"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	dem "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
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
	health      []float32
}

var trMap = make(map[int]*playerMapping)
var ctMap = make(map[int]*playerMapping)

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
	snapshotCollectionSize := 0
	fullMap := utils.AnnotatedMap{IconsList: nil, SourceMap: header.MapName}
	mapMetadata := metadata.MapNameToMap[header.MapName]
	imageIndex := 0
	p.RegisterEventHandler(func(e events.FrameDone) {
		gs := p.GameState()
		if !(gs == nil) && roundDir != dirName {
			processFrameEnd(gs, p, &fullMap, mapMetadata, &imageIndex, roundDir, &snapshotCollectionSize)
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
		trMap, ctMap = remakePlayerMappings(gs)
		newScore := "ct_" + strconv.Itoa(gs.TeamCounterTerrorists().Score()) +
			"_t_" + strconv.Itoa(gs.TeamTerrorists().Score())
		if gs.TeamTerrorists().Score() >= 6 && gs.TeamCounterTerrorists().Score() >= 10 {
			fmt.Println(gs.TeamTerrorists().Score(), gs.TeamCounterTerrorists().Score())
		}
		roundDir = dirName + "/" + newScore
		dirExists, _ := exists(roundDir)
		imageIndex = 0
		snapshotCollectionSize = 0
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

			//csvData := organizeTabularData(trMap, ctMap, snapshotCollectionSize)
			//writeToCSV(csvData, roundDir+"/tabular.csv")
		}

	})
	err = p.ParseToEnd()
	p.Close()
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

func writeToCSV(data [][]string, filePath string) {
	file, err := os.Create(filePath)
	checkError(err)
	defer file.Close()

	writer := csv.NewWriter(file)

	err = writer.WriteAll(data)
	checkError(err)
}

func organizeTabularData(trMap map[int]*playerMapping,
	ctMap map[int]*playerMapping, snapshotCollectionSize int) [][]string {

	var csvData [][]string
	var singleRow [10]string
	for countSnapshot := 0; countSnapshot < snapshotCollectionSize; countSnapshot++ {
		for _, playerData := range trMap {
			singleRow[playerData.playerSeqID] = strconv.FormatFloat(float64(playerData.health[countSnapshot]), 'f', -1, 32)
		}
		for _, playerData := range ctMap {
			singleRow[5+playerData.playerSeqID] = strconv.FormatFloat(float64(playerData.health[countSnapshot]), 'f', -1, 32)
		}

		csvData = append(csvData, singleRow[:])
	}
	return csvData
}

func processTeamHP(members []*common.Player, teamMap map[int]*playerMapping) {
	for _, t := range members {
		fmt.Println(t.UserID, teamMap[t.UserID])
		teamMap[t.UserID].health = append(teamMap[t.UserID].health, 0.0)
	}
}

func processPlayerHP(gs dem.GameState, fullMap *utils.AnnotatedMap,
	trMap map[int]*playerMapping, ctMap map[int]*playerMapping) {
	//add t icons
	processTeamHP(gs.TeamTerrorists().Members(), trMap)
	//add ct icons
	processTeamHP(gs.TeamCounterTerrorists().Members(), ctMap)

}

func remakeTeamMapping(members []*common.Player) map[int]*playerMapping {
	teamMap := make(map[int]*playerMapping)
	for _, player := range members {

		teamMap[player.UserID] = &playerMapping{playerSeqID: 0, health: nil}
	}
	seqID := 0
	var keys []int
	for k := range teamMap {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		teamMap[k].playerSeqID = seqID
		seqID++
	}
	return teamMap
}

func remakePlayerMappings(gs dem.GameState) (map[int]*playerMapping, map[int]*playerMapping) {

	trMap := remakeTeamMapping(gs.TeamTerrorists().Members())
	ctMap := remakeTeamMapping(gs.TeamCounterTerrorists().Members())
	return trMap, ctMap
}

func processPlayerInformation(gs dem.GameState, fullMap *utils.AnnotatedMap,
	mapMetadata metadata.Map, trMap map[int]*playerMapping, ctMap map[int]*playerMapping) {

	processPlayerPositions(gs, fullMap, mapMetadata)
	//processPlayerHP(gs, fullMap, trMap, ctMap)
}

func processFrameEnd(gs dem.GameState, p dem.Parser,
	fullMap *utils.AnnotatedMap, mapMetadata metadata.Map, imageIndex *int,
	roundDir string, snapshotCollectionSize *int) {
	currentTime := getCurrentTime(p)
	currentRoundTime := getRoundTime(p)
	if currentRoundTime%posUpdateInterval == 0 && currentTime != lastUpdate {
		lastUpdate = currentTime

		processPlayerInformation(gs, fullMap, mapMetadata, trMap, ctMap)
		generateMap(fullMap, roundDir, imageIndex)
		*snapshotCollectionSize++
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
