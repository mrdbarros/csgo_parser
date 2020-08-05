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
	"github.com/nfnt/resize"
)

var currentState = ""
var gameReset = false
var gameStarted = false
var discretizeFactor = 20.0
var roundStartTime float64
var lastUpdate = 0.0
var tickRate = 0
var updateInterval = 2.0

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
	winTeamCurrentRound := "t"
	roundDir := dirName
	snapshotCollectionSize := 0
	mapMetadata := metadata.MapNameToMap[header.MapName]
	imageIndex := 0
	isNewRound := false
	var smokeList []events.GrenadeEvent
	var incendiaryList []events.GrenadeEvent
	roundFreezeTime := false
	bombPlanted := false
	p.RegisterEventHandler(func(e events.FrameDone) {
		gs := p.GameState()
		if !(gs == nil) && roundDir != dirName && !roundFreezeTime {
			processFrameEnd(gs, header.MapName, p, mapMetadata, &imageIndex,
				roundDir, &snapshotCollectionSize, roundCSVPath, &isNewRound, bombPlanted, smokeList, incendiaryList)
		}

	})

	p.RegisterEventHandler(func(e events.RoundFreezetimeEnd) {
		roundFreezeTime = false

	})

	p.RegisterEventHandler(func(e events.BombPlanted) {
		bombPlanted = true

	})

	p.RegisterEventHandler(func(e events.SmokeStart) {
		smokeList = append(smokeList, e.GrenadeEvent)

	})

	p.RegisterEventHandler(func(e events.SmokeExpired) {
		for i, smokeEvent := range smokeList {
			if e.GrenadeEvent.GrenadeEntityID == smokeEvent.GrenadeEntityID {
				// Remove the element at index i from a.
				smokeList[i] = smokeList[len(smokeList)-1]          // Copy last element to index i.
				smokeList[len(smokeList)-1] = events.GrenadeEvent{} // Erase last element.
				smokeList = smokeList[:len(smokeList)-1]
			}
		}

	})

	p.RegisterEventHandler(func(e events.FireGrenadeStart) {
		incendiaryList = append(incendiaryList, e.GrenadeEvent)

	})

	p.RegisterEventHandler(func(e events.FireGrenadeExpired) {
		for i, incendiaryEvent := range incendiaryList {
			if e.GrenadeEvent.GrenadeEntityID == incendiaryEvent.GrenadeEntityID {
				// Remove the element at index i from a.
				incendiaryList[i] = incendiaryList[len(incendiaryList)-1]     // Copy last element to index i.
				incendiaryList[len(incendiaryList)-1] = events.GrenadeEvent{} // Erase last element.
				incendiaryList = incendiaryList[:len(incendiaryList)-1]
			}
		}

	})

	p.RegisterEventHandler(func(e events.RoundStart) {
		gs := p.GameState()
		roundFreezeTime = true
		bombPlanted = false
		smokeList = []events.GrenadeEvent{}
		incendiaryList = []events.GrenadeEvent{}

		allPlayers = remakePlayerMappings(gs)
		newScore := "ct_" + utils.PadLeft(strconv.Itoa(gs.TeamCounterTerrorists().Score()), "0", 2) +
			"_t_" + utils.PadLeft(strconv.Itoa(gs.TeamTerrorists().Score()), "0", 2)

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
		isNewRound = true
		roundCSV, err := os.Create(roundCSVPath)
		checkError(err)
		defer roundCSV.Close()

		roundStartTime = getCurrentTime(p)
		lastUpdate = 0.0
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
	//var waitGroup sync.WaitGroup
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

func processPlayersHPAndFlash(allPlayers map[int]*playerMapping, sortedUserIDs []int) (newCSVRow []string, header []string) {

	newCSVRow = []string{
		"0", "0", "0", "0", "0", "0", "0", "0", "0", "0",
		"0", "0", "0", "0", "0", "0", "0", "0", "0", "0"}
	tCount := 0
	ctCount := 0
	playerBasePos := 0
	for _, userID := range sortedUserIDs {
		if _, ok := allPlayers[userID]; ok {
			player := allPlayers[userID].playerObject

			isCT := (player.Team == 3)
			isTR := (player.Team == 2)

			if !isCT && tCount > 4 || isCT && ctCount > 4 {
				fmt.Println("invalid team size")
				break
			}

			if !(isCT || isTR) {
				fmt.Println("invalid team")
				break
			}

			if isCT {
				playerBasePos = 5 + ctCount
				ctCount++
			}
			if isTR {
				playerBasePos = tCount
				tCount++
			}

			newCSVRow[playerBasePos] = strconv.FormatFloat(float64(player.Health())/100, 'f', -1, 32)
			newCSVRow[10+playerBasePos] = strconv.FormatFloat(player.FlashDurationTimeRemaining().Seconds(), 'f', -1, 32)

		} else {
			fmt.Println("key not found", userID)
		}

	}
	header = []string{"t_1", "t_2", "t_3", "t_4", "t_5", "ct_1", "ct_2", "ct_3", "ct_4", "ct_5",
		"t_1_blindtime", "t_2_blindtime", "t_3_blindtime", "t_4_blindtime", "t_5_blindtime",
		"ct_1_blindtime", "ct_2_blindtime", "ct_3_blindtime", "ct_4_blindtime", "ct_5_blindtime"}
	return newCSVRow[:], header
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

func findIntInSlice(slice []int, number int) bool {
	for _, sliceNumber := range slice {
		if sliceNumber == number {
			return true
		}
	}
	return false
}

func fillPlayerWeapons(player *common.Player) []string {
	//"mainweapon", "secweapon", "flashbangs", "hassmoke", "hasmolotov", "hashe","armorvalue","hashelmet","hasdefusekit/hasc4",

	weapons := []string{"0", "0", "0", "0", "0", "0", "0", "0", "0"}

	primaryWeaponClasses := []int{2, 3, 4}
	secondaryWeaponClasses := []int{1}

	molotovAndIncendiary := []int{502, 503}

	equipSlice := player.Weapons()
	equipClass := 0
	equipType := 0
	for _, equip := range equipSlice {
		equipClass = int(equip.Class())
		equipType = int(equip.Type)
		if findIntInSlice(primaryWeaponClasses, equipClass) {
			weapons[0] = strconv.Itoa(equipType)
		}
		if findIntInSlice(secondaryWeaponClasses, equipClass) {
			weapons[1] = strconv.Itoa(equipType)
		}
		if equipType == 504 { //flash
			weapons[2] = strconv.Itoa(player.AmmoLeft[equip.AmmoType()])
		}
		if equipType == 505 { //smoke
			weapons[3] = "1"
		}
		if findIntInSlice(molotovAndIncendiary, equipType) { //molotov or incendiary
			weapons[4] = "1"
		}
		if equipType == 506 { //HE
			weapons[5] = "1"
		}
		if equipType == 406 || equipType == 404 { //defuse kit / c4
			weapons[8] = "1"
		}

	}
	weapons[6] = strconv.Itoa(player.Armor())
	if player.HasHelmet() {
		weapons[7] = "1"
	}
	return weapons
}

func processPlayerWeapons(allPlayers map[int]*playerMapping, sortedUserIDs []int) (newCSVRow []string, header []string) {
	newCSVRow = []string{
		"0", "0", "0", "0", "0", "0", "0", "0", "0",
		"0", "0", "0", "0", "0", "0", "0", "0", "0",
		"0", "0", "0", "0", "0", "0", "0", "0", "0",
		"0", "0", "0", "0", "0", "0", "0", "0", "0",
		"0", "0", "0", "0", "0", "0", "0", "0", "0",
		"0", "0", "0", "0", "0", "0", "0", "0", "0",
		"0", "0", "0", "0", "0", "0", "0", "0", "0",
		"0", "0", "0", "0", "0", "0", "0", "0", "0",
		"0", "0", "0", "0", "0", "0", "0", "0", "0",
		"0", "0", "0", "0", "0", "0", "0", "0", "0"}
	tCount := 0
	ctCount := 0
	lenPerPlayer := len(newCSVRow) / 10
	playerInfo := []string{}
	playerBasePos := 0
	for _, userID := range sortedUserIDs {
		if _, ok := allPlayers[userID]; ok {
			player := allPlayers[userID].playerObject
			playerInfo = fillPlayerWeapons(player)

			isCT := (player.Team == 3)
			isTR := (player.Team == 2)

			if !isCT && tCount > 4 || isCT && ctCount > 4 {
				fmt.Println("invalid team size")
				break
			}

			if !(isCT || isTR) {
				fmt.Println("invalid team")
				break
			}

			if isCT {
				playerBasePos = 5*lenPerPlayer + ctCount*lenPerPlayer
				ctCount++
			} else if isTR {
				playerBasePos = tCount * lenPerPlayer
				tCount++
			}

			for i, info := range playerInfo {
				newCSVRow[playerBasePos+i] = info
			}

		} else {
			fmt.Println("key not found", userID)
		}

	}
	header = []string{
		"t_1_mainweapon", "t_1_secweapon", "t_1_flashbangs", "t_1_hassmoke", "t_1_hasmolotov", "t_1_hashe", "t_1_armor", "t_1_hashelmet", "t_1_hasc4",
		"t_2_mainweapon", "t_2_secweapon", "t_2_flashbangs", "t_2_hassmoke", "t_2_hasmolotov", "t_2_hashe", "t_2_armor", "t_2_hashelmet", "t_2_hasc4",
		"t_3_mainweapon", "t_3_secweapon", "t_3_flashbangs", "t_3_hassmoke", "t_3_hasmolotov", "t_3_hashe", "t_3_armor", "t_3_hashelmet", "t_3_hasc4",
		"t_4_mainweapon", "t_4_secweapon", "t_4_flashbangs", "t_4_hassmoke", "t_4_hasmolotov", "t_4_hashe", "t_4_armor", "t_4_hashelmet", "t_4_hasc4",
		"t_5_mainweapon", "t_5_secweapon", "t_5_flashbangs", "t_5_hassmoke", "t_5_hasmolotov", "t_5_hashe", "t_5_armor", "t_5_hashelmet", "t_5_hasc4",
		"ct_1_mainweapon", "ct_1_secweapon", "ct_1_flashbangs", "ct_1_hassmoke", "ct_1_hasmolotov", "ct_1_hashe", "ct_1_armor", "ct_1_hashelmet", "ct_1_hasdefusekit",
		"ct_2_mainweapon", "ct_2_secweapon", "ct_2_flashbangs", "ct_2_hassmoke", "ct_2_hasmolotov", "ct_2_hashe", "ct_2_armor", "ct_2_hashelmet", "ct_2_hasdefusekit",
		"ct_3_mainweapon", "ct_3_secweapon", "ct_3_flashbangs", "ct_3_hassmoke", "ct_3_hasmolotov", "ct_3_hashe", "ct_3_armor", "ct_3_hashelmet", "ct_3_hasdefusekit",
		"ct_4_mainweapon", "ct_4_secweapon", "ct_4_flashbangs", "ct_4_hassmoke", "ct_4_hasmolotov", "ct_4_hashe", "ct_4_armor", "ct_4_hashelmet", "ct_4_hasdefusekit",
		"ct_5_mainweapon", "ct_5_secweapon", "ct_5_flashbangs", "ct_5_hassmoke", "ct_5_hasmolotov", "ct_5_hashe", "ct_5_armor", "ct_5_hashelmet", "ct_5_hasdefusekit"}
	return newCSVRow[:], header
}

func processPlayerInformation(fullMap *utils.AnnotatedMap,
	mapMetadata metadata.Map, allPlayers map[int]*playerMapping) (newCSVRow []string, newHeader []string) {

	sortedUserIDs := sortPlayersByUserID(allPlayers)
	processPlayerPositions(allPlayers, fullMap, mapMetadata, sortedUserIDs)

	tempCSV, tempHeader := processPlayersHPAndFlash(allPlayers, sortedUserIDs)
	newCSVRow = append(newCSVRow, tempCSV...)
	newHeader = append(newHeader, tempHeader...)

	tempCSV, tempHeader = processPlayerWeapons(allPlayers, sortedUserIDs)
	newCSVRow = append(newCSVRow, tempCSV...)
	newHeader = append(newHeader, tempHeader...)

	return newCSVRow, newHeader
}

func processOtherGameInfo(gs dem.GameState, fullMap *utils.AnnotatedMap, mapMetadata metadata.Map, bombPlanted bool, currentRoundTime float64,
	smokeList []events.GrenadeEvent, incendiaryList []events.GrenadeEvent) (newCSVRow []string, header []string) {
	newCSVRow = []string{"0"}
	if bombPlanted {
		bombPosition := gs.Bomb().Position()
		x, y := mapMetadata.TranslateScale(bombPosition.X, bombPosition.Y)
		newIcon := utils.Icon{X: x - 10, Y: y - 10, IconName: "bomb_planted"}
		(*fullMap).IconsList = append((*fullMap).IconsList, newIcon)
	}

	processGrenadesPositions(fullMap, mapMetadata, smokeList, incendiaryList)

	newCSVRow[0] = strconv.FormatFloat(currentRoundTime, 'f', -1, 32)
	header = []string{"round_time"}
	return newCSVRow, header
}

func processFrameEnd(gs dem.GameState, mapName string, p dem.Parser, mapMetadata metadata.Map, imageIndex *int,
	roundDir string, snapshotCollectionSize *int, roundCSVPath string, isNewRound *bool, bombPlanted bool, smokeList []events.GrenadeEvent,
	incendiaryList []events.GrenadeEvent) {

	currentRoundTime := getRoundTime(p)
	if (currentRoundTime - lastUpdate) > updateInterval {
		lastUpdate = currentRoundTime

		writeData := [][]string{}
		fullMap := utils.AnnotatedMap{IconsList: nil, SourceMap: mapName}
		newCSVRow := []string{}
		newHeader := []string{}
		tempCSV, tempHeader := processPlayerInformation(&fullMap, mapMetadata, allPlayers)
		newCSVRow = append(newCSVRow, tempCSV...)
		newHeader = append(newHeader, tempHeader...)
		tempCSV, tempHeader = processOtherGameInfo(gs, &fullMap, mapMetadata, bombPlanted, currentRoundTime, smokeList, incendiaryList)
		newCSVRow = append(newCSVRow, tempCSV...)
		newHeader = append(newHeader, tempHeader...)
		generateMap(&fullMap, roundDir, imageIndex)

		if *isNewRound {
			writeData = append(writeData, newHeader)
			writeData = append(writeData, newCSVRow)
			writeToCSV(writeData, roundCSVPath)
			*isNewRound = false
		} else {
			writeData = append(writeData, newCSVRow)
			writeToCSV(writeData, roundCSVPath)
		}

		*snapshotCollectionSize++
	}

}

func getRoundTime(p dem.Parser) float64 {
	return getCurrentTime(p) - roundStartTime
}

func getCurrentTime(p dem.Parser) float64 {
	currentFrame := p.CurrentFrame()
	return float64(currentFrame) / float64(tickRate)
}

func generateMap(fullMap *utils.AnnotatedMap, roundDir string, imageIndex *int) {
	img_original := utils.DrawMap(*fullMap)
	img := resize.Resize(800, 0, img_original, resize.Bilinear)
	third, err := os.Create(roundDir + "/output_map" +
		utils.PadLeft(strconv.Itoa(*imageIndex), "0", 2) + ".jpg")
	if err != nil {
		log.Fatalf("failed to create: %s", err)
	}
	err = jpeg.Encode(third, img, &jpeg.Options{jpeg.DefaultQuality})
	checkError(err)
	*imageIndex++
	third.Close()
}

func processGrenadesPositions(fullMap *utils.AnnotatedMap, mapMetadata metadata.Map, smokeList []events.GrenadeEvent, incendiaryList []events.GrenadeEvent) {
	//add incendiary icons

	for _, incendiary := range incendiaryList {
		x, y := mapMetadata.TranslateScale(incendiary.Position.X, incendiary.Position.Y)
		newIcon := utils.Icon{X: x - 10, Y: y - 10, IconName: "incendiary"}
		(*fullMap).IconsList = append((*fullMap).IconsList, newIcon)
	}
	//add smoke icons
	for _, smoke := range smokeList {
		x, y := mapMetadata.TranslateScale(smoke.Position.X, smoke.Position.Y)
		newIcon := utils.Icon{X: x - 10, Y: y - 10, IconName: "smoke"}
		(*fullMap).IconsList = append((*fullMap).IconsList, newIcon)
	}
}

func processPlayerPositions(allPlayers map[int]*playerMapping, fullMap *utils.AnnotatedMap,
	mapMetadata metadata.Map, sortedUserIDs []int) {

	//add players icons
	tCount := 0
	ctCount := 0
	playerCount := 0
	for _, userID := range sortedUserIDs {
		if _, ok := allPlayers[userID]; ok {
			player := allPlayers[userID].playerObject

			isCT := (player.Team == 3)
			isTR := (player.Team == 2)

			if !isCT && tCount > 4 || isCT && ctCount > 4 {
				fmt.Println("invalid team size")
				break
			}

			if !(isCT || isTR) {
				fmt.Println("invalid team")
				break
			}

			x, y := mapMetadata.TranslateScale(player.Position().X, player.Position().Y)
			var icon string
			if isTR { //terrorist
				icon = "terrorist_1"
				tCount++
				playerCount = tCount
			} else if isCT {
				icon = "ct_1"
				ctCount++
				playerCount = ctCount
			}
			newIcon := utils.Icon{X: x - 10, Y: y - 10, IconName: icon} //t or ct icon
			(*fullMap).IconsList = append((*fullMap).IconsList, newIcon)
			newIcon = utils.Icon{X: x - 10, Y: y - 10, IconName: strconv.Itoa(playerCount)}
			(*fullMap).IconsList = append((*fullMap).IconsList, newIcon)
		} else {
			fmt.Println("key not found", userID)
		}

	}

}
