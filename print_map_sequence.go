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
	playerSeqID  int
	playerObject *common.Player
}

var allPlayers = make(map[int]*playerMapping)

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

	var roundCSVPath string
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
			processFrameEnd(gs, p, &fullMap, mapMetadata, &imageIndex,
				roundDir, &snapshotCollectionSize, roundCSVPath)
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
		allPlayers = remakePlayerMappings(gs)
		newScore := "ct_" + strconv.Itoa(gs.TeamCounterTerrorists().Score()) +
			"_t_" + strconv.Itoa(gs.TeamTerrorists().Score())

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
		roundCSVPath = roundDir + "/tabular.csv"
		roundCSV, err := os.Create(roundCSVPath)
		checkError(err)
		defer roundCSV.Close()
		roundStartTime = getCurrentTime(p)
	})

	p.RegisterEventHandler(func(e events.RoundEnd) {

		winTeam := e.Winner
		if winTeam == 2 {
			winTeamCurrentRound = "t"
		} else if winTeam == 3 {
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
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	checkError(err)
	writer := csv.NewWriter(file)

	err = writer.WriteAll(data)
	checkError(err)
}

// func organizeTabularData(allPlayers map[int]*playerMapping,
// 	snapshotCollectionSize int) [][]string {

// 	var csvData [][]string
// 	var singleRow [10]string
// 	for countSnapshot := 0; countSnapshot < snapshotCollectionSize; countSnapshot++ {
// 		ctCount, tCount := 0, 0
// 		for _, playerData := range allPlayers {
// 			if playerData.playerObject.Team == 2 { //terrorist
// 				singleRow[tCount] = strconv.FormatFloat(float64(playerData.playerObject.Health(),
// 				'f',-1,32)

// 			} else if playerData.playerObject.Team == 3 { //ct

// 			}
// 			singleRow[playerData.playerSeqID] = strconv.FormatFloat(float64(playerData.health), 'f', -1, 32)
// 		}

// 		csvData = append(csvData, singleRow[:])
// 	}
// 	return csvData
// }

// func processTeamHP(gs dem.GameState, members []*common.Player, teamMap map[int]*playerMapping) {
// 	for _, t := range members {
// 		//fmt.Println(t.UserID, teamMap[t.UserID])
// 		if _, ok := teamMap[t.UserID]; ok {
// 			teamMap[t.UserID].health = float32(t.Health()) / 100
// 		} else {
// 			fmt.Println("key", t.UserID, "not found")

// 		}

// 	}
// }

func processPlayersHP(fullMap *utils.AnnotatedMap,
	allPlayers map[int]*playerMapping, roundCSVPath string, sortedUserIDs []int) {

	var dataCSV [][]string
	newCSVRow := [10]string{"0", "0", "0", "0", "0", "0", "0", "0", "0", "0"}
	tCount := 0
	ctCount := 0
	for _, userID := range sortedUserIDs {
		if _, ok := allPlayers[userID]; ok {
			player := allPlayers[userID].playerObject
			if player.Team == 2 { //terrorist
				newCSVRow[tCount] = strconv.FormatFloat(float64(player.Health())/100, 'f', -1, 32)
				tCount++
			} else if player.Team == 3 { //ct
				newCSVRow[5+ctCount] = strconv.FormatFloat(float64(player.Health())/100, 'f', -1, 32)
				ctCount++
			}
		} else {
			fmt.Println("key not found", userID)
		}

	}
	dataCSV = append(dataCSV, newCSVRow[:])
	writeToCSV(dataCSV, roundCSVPath)
}

func remakePlayerMappings(gs dem.GameState) map[int]*playerMapping {
	newAllPlayers := make(map[int]*playerMapping)
	players := gs.Participants().Playing()
	for _, player := range players {
		newAllPlayers[player.UserID] = &playerMapping{playerSeqID: 0, playerObject: player}
	}
	seqID := 0
	var keys []int
	for k := range newAllPlayers {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		newAllPlayers[k].playerSeqID = seqID
		seqID++
	}
	return newAllPlayers
}

func sortPlayersByUserID(allPlayers map[int]*playerMapping) []int {

	var keys []int
	for userID := range allPlayers {
		keys = append(keys, userID)
	}
	sort.Ints(keys)
	return keys
}

func processPlayerInformation(fullMap *utils.AnnotatedMap,
	mapMetadata metadata.Map, allPlayers map[int]*playerMapping,
	roundCSVPath string) {

	sortedUserIDs := sortPlayersByUserID(allPlayers)
	processPlayerPositions(allPlayers, fullMap, mapMetadata, sortedUserIDs)
	processPlayersHP(fullMap, allPlayers, roundCSVPath, sortedUserIDs)
}

func processFrameEnd(gs dem.GameState, p dem.Parser,
	fullMap *utils.AnnotatedMap, mapMetadata metadata.Map, imageIndex *int,
	roundDir string, snapshotCollectionSize *int, roundCSVPath string) {
	currentTime := getCurrentTime(p)
	currentRoundTime := getRoundTime(p)
	if currentRoundTime%posUpdateInterval == 0 && currentTime != lastUpdate {
		lastUpdate = currentTime

		processPlayerInformation(fullMap, mapMetadata, allPlayers, roundCSVPath)
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

func processPlayerPositions(allPlayers map[int]*playerMapping, fullMap *utils.AnnotatedMap,
	mapMetadata metadata.Map, sortedUserIDs []int) {

	(*fullMap).IconsList = nil
	//add players icons
	for _, userID := range sortedUserIDs {
		if _, ok := allPlayers[userID]; ok {
			player := allPlayers[userID].playerObject
			x, y := mapMetadata.TranslateScale(player.Position().X, player.Position().Y)
			var icon string
			if player.Team == 2 { //terrorist
				icon = "terrorist_1"
			} else {
				icon = "ct_1"
			}
			newIcon := utils.Icon{X: x - 10, Y: y - 10, IconName: icon}
			(*fullMap).IconsList = append((*fullMap).IconsList, newIcon)
		} else {
			fmt.Println("key not found", userID)
		}

	}

}
