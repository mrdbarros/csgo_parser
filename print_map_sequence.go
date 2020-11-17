package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	dp "github.com/markus-wa/godispatch"

	dem "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
	metadata "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/metadata"
	utils "github.com/mrdbarros/csgo_analyze/utils"
	"github.com/nfnt/resize"
)

type playerMapping struct {
	playerObject *common.Player
}

type CompositeEventHandler interface {
	Register() error
	Unregister() error
	SetConfig(parser *dem.Parser, tickRate int, mapMetadata metadata.Map) error
}

type IconGenerator interface {
	GetIcons() ([]utils.Icon, error)
}

type TabularGenerator interface {
	GetTabularData() ([]string, []string, error) //header, data, error
}

type StatGenerator interface {
	GetStatistics() ([]string, []string, error) //header, data, error
}

type basicHandler struct {
	parser           *dem.Parser
	tickRate         int
	mapMetadata      metadata.Map
	roundStartTime   float64
	rootPath         string
	roundDirPath     string
	roundWinner      string
	roundTabularPath string
	roundStatPath    string
	roundNumber      int
}

type poppingGrenadeHandler struct {
	basicHandler          *basicHandler
	activeGrenades        []*grenadeTracker
	grenadeStartHandlerID dp.HandlerIdentifier
	baseIcons             map[common.EquipmentType]utils.Icon
}

type poppingGrenadeManager struct {
	popHandler *poppingGrenadeHandler
}

//e holds smoke start/expired or inferno start/expired and other grenade events
func (ph *poppingGrenadeHandler) GrenadeStartHandler(e events.GrenadeEventIf) {

	// if molly, incgrenade or smoke
	if e.Base().GrenadeType == common.EqSmoke || e.Base().GrenadeType == common.EqIncendiary || e.Base().GrenadeType == common.EqMolotov {
		parser := *(ph.basicHandler.parser)
		eventTime := getCurrentTime(parser, ph.basicHandler.tickRate)
		grenadeEntityID := e.Base().GrenadeEntityID
		if ph.IsTracked(grenadeEntityID) {
			ph.RemoveGrenade(grenadeEntityID)
		} else {
			newGrenade := grenadeTracker{grenadeEvent: e.Base(), grenadeTime: eventTime}
			ph.activeGrenades = append(ph.activeGrenades, &newGrenade)
		}

	}

}

func (ph *poppingGrenadeHandler) IsTracked(entityID int) bool {
	for _, activeGrenade := range ph.activeGrenades {
		if activeGrenade.grenadeEvent.GrenadeEntityID == entityID {
			return true
		}

	}
	return false
}

func (ph *poppingGrenadeHandler) Register() error {
	parser := *(ph.basicHandler.parser)
	ph.grenadeStartHandlerID = parser.RegisterEventHandler(ph.GrenadeStartHandler)
	return nil
}

func (ph *poppingGrenadeHandler) Unregister() error {
	parser := *(ph.basicHandler.parser)
	parser.UnregisterEventHandler(ph.grenadeStartHandlerID)
	return nil

}

func (ph poppingGrenadeManager) GetIcons() ([]utils.Icon, error) {
	var iconList []utils.Icon
	popHandler := ph.popHandler
	for _, activeGrenade := range popHandler.activeGrenades {
		newIcon := popHandler.baseIcons[activeGrenade.grenadeEvent.GrenadeType]
		x, y := popHandler.basicHandler.mapMetadata.TranslateScale(activeGrenade.grenadeEvent.Position.X, activeGrenade.grenadeEvent.Position.Y)
		newIcon.X, newIcon.Y = x, y
		iconList = append(iconList, newIcon)
	}
	return iconList, nil
}

func (bh *basicHandler) SetConfig(parser *dem.Parser, tickRate int, mapMetadata metadata.Map) error {
	bh.parser = parser
	bh.tickRate = tickRate
	bh.mapMetadata = mapMetadata
	return nil
}

func (ph *poppingGrenadeHandler) RemoveGrenade(entityID int) {

	for i, grenade := range ph.activeGrenades {
		if grenade.grenadeEvent.GrenadeEntityID == entityID {
			ph.activeGrenades[i] = ph.activeGrenades[len(ph.activeGrenades)-1]
			ph.activeGrenades = ph.activeGrenades[:(len(ph.activeGrenades) - 1)]
			break
		}
	}
}

func (ph *poppingGrenadeHandler) SetBaseIcons() {
	ph.baseIcons = map[common.EquipmentType]utils.Icon{
		505: utils.Icon{IconName: utils.SmokeIconName},
		502: utils.Icon{IconName: utils.IncendiaryIconName},
		503: utils.Icon{IconName: utils.IncendiaryIconName},
	}
}

// type gameStateInfoHandler struct {

// 	bombPlanted          bool
// 	bombPlantedTime      float64
// 	bombPlantedHandlerID dp.HandlerIdentifier
// 	roundStartHandlerID  dp.HandlerIdentifier
// 	baseIcons            map[string]utils.Icon
// }

type gameStateInfoManager struct {
	basicHandler *basicHandler
}

func (gm gameStateInfoManager) GetTabularData() ([]string, []string, error) {
	parser := *(gm.basicHandler.parser)
	gs := parser.GameState()
	newCSVRow := []string{"0"}
	currentRoundTime := getCurrentTime(parser, gm.basicHandler.tickRate)

	newCSVRow[0] = strconv.FormatFloat(currentRoundTime-gm.basicHandler.roundStartTime, 'f', -1, 32)
	header := []string{"round_time"}
	return newCSVRow, header, nil

}

type playerInfoHandler struct {
	basicHandler        *basicHandler
	playerMappings      map[int]*playerMapping
	sortedUserIDs       []int
	roundStartHandlerID dp.HandlerIdentifier
}

type playerInfoManager struct {
	playerInfoHandler *playerInfoHandler
}

func (ph *playerInfoHandler) Register() error {
	parser := *(ph.basicHandler.parser)
	ph.roundStartHandlerID = parser.RegisterEventHandler(ph.RoundStartHandler)
	return nil
}

func (ph *playerInfoHandler) RoundStartHandler(e events.RoundStart) {
	parser := *(ph.basicHandler.parser)
	ph.playerMappings = remakePlayerMappings(parser.GameState())
	ph.sortedUserIDs = sortPlayersByUserID(ph.playerMappings)

}
func (ph *playerInfoHandler) processPlayerPositions() (iconList []utils.Icon) {
	ctCount := 0
	tCount := 0
	playerCount := 0
	for _, userID := range ph.sortedUserIDs {
		if playerMap, ok := ph.playerMappings[userID]; ok {
			player := playerMap.playerObject
			isCT := (player.Team == 3)
			isTR := (player.Team == 2)

			if player.Health() > 0 {
				x, y := ph.basicHandler.mapMetadata.TranslateScale(player.Position().X, player.Position().Y)
				var icon string

				if isCT {
					playerCount = 5 + ctCount
					icon = "ct_"
					if player.HasDefuseKit() {
						newIcon := utils.Icon{X: x, Y: y, IconName: "kit"} //t or ct icon
						iconList = append(iconList, newIcon)
					}
					ctCount++
				} else if isTR {
					icon = "terrorist_"
					playerCount = tCount
					tCount++
				}

				newIcon := utils.Icon{X: x, Y: y, IconName: icon + strconv.Itoa(playerCount), Rotate: float64(player.ViewDirectionX())} //t or ct icon
				iconList = append(iconList, newIcon)
				newIcon = utils.Icon{X: x, Y: y, IconName: strconv.Itoa(playerCount)}
				iconList = append(iconList, newIcon)

			}

		} else {
			fmt.Println("key not found", userID)
		}

	}
	return iconList
}

func (ph playerInfoHandler) processPlayerWeapons() (newCSVRow []string, header []string) {

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
	lenPerPlayer := len(newCSVRow) / 10
	playerInfo := []string{}
	playerBasePos := 0
	ctCount := 0
	tCount := 0
	playerCount := 0
	for _, userID := range ph.sortedUserIDs {
		if playerMap, ok := allPlayers[userID]; ok {
			player := playerMap.playerObject
			isCT := (player.Team == 3)
			isTR := (player.Team == 2)
			playerInfo = fillPlayerWeapons(player)

			if isCT {
				playerBasePos = lenPerPlayer * (5 + ctCount)

				ctCount++
			} else if isTR {
				playerBasePos = lenPerPlayer * tCount
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

func (ph *playerInfoHandler) getPlayersHPAndFlashtime() (newCSVRow []string, header []string) {

	newCSVRow = []string{
		"0", "0", "0", "0", "0", "0", "0", "0", "0", "0",
		"0", "0", "0", "0", "0", "0", "0", "0", "0", "0"}
	tCount := 0
	ctCount := 0
	playerBasePos := 0
	for _, userID := range ph.sortedUserIDs {
		if _, ok := ph.playerMappings[userID]; ok {
			player := ph.playerMappings[userID].playerObject

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

func (pm playerInfoManager) GetTabularData() (newHeader []string, newCSVRow []string, err error) {

	tempCSV, tempHeader := pm.playerInfoHandler.getPlayersHPAndFlashtime()
	newCSVRow = append(newCSVRow, tempCSV...)
	newHeader = append(newHeader, tempHeader...)

	tempCSV, tempHeader = pm.playerInfoHandler.processPlayerWeapons()
	newCSVRow = append(newCSVRow, tempCSV...)
	newHeader = append(newHeader, tempHeader...)
	return newHeader, newCSVRow, err
}

func (pm playerInfoManager) GetIcons() ([]utils.Icon, error) {

	return pm.playerInfoHandler.processPlayerPositions(), nil

}

type bombHandler struct {
	basicHandler         *basicHandler
	bombPlanted          bool
	bombPlantedTime      float64
	bombPlantedHandlerID dp.HandlerIdentifier
	roundStartHandlerID  dp.HandlerIdentifier
	baseIcons            map[string]utils.Icon
}

func (bh *bombHandler) Register() error {
	parser := *(bh.basicHandler.parser)
	bh.bombPlantedHandlerID = parser.RegisterEventHandler(bh.BombPlantedHandler)
	bh.roundStartHandlerID = parser.RegisterEventHandler(bh.RoundStartHandler)
	return nil
}

func (bh *bombHandler) BombPlantedHandler(e events.BombPlanted) {
	parser := (*bh.basicHandler.parser)
	bh.bombPlanted = true
	bh.bombPlantedTime = getCurrentTime(parser, bh.basicHandler.tickRate)

}

func (bh *bombHandler) RoundStartHandler(e events.RoundStart) {

	bh.bombPlanted = false

}

type bombManager struct {
	bmbHandler *bombHandler
}

func (bm bombManager) GetIcons() (icons []utils.Icon, err error) {
	bh := bm.bmbHandler
	parser := (*bh.basicHandler.parser)
	bomb := parser.GameState().Bomb()
	var icon string
	if bh.bombPlanted {
		icon = "bomb_planted"
	} else if bomb.Carrier == nil {
		icon = "bomb_dropped"
	} else {
		icon = "c4_carrier"
	}
	bombPosition := bomb.Position()
	x, y := bh.basicHandler.mapMetadata.TranslateScale(bombPosition.X, bombPosition.Y)
	icons = append(icons, utils.Icon{IconName: icon, X: x, Y: y})
	return icons, nil
}

func (bm bombManager) GetTabularData() ([]string, []string, error) {
	newCSVRow := []string{"0"}
	if bm.bmbHandler.bombPlanted {
		parser := (*bm.bmbHandler.basicHandler.parser)
		currentTime := getCurrentTime(parser, bm.bmbHandler.basicHandler.tickRate)
		newCSVRow[0] = strconv.FormatFloat(currentTime-bm.bmbHandler.bombPlantedTime, 'f', -1, 32)
	}

	header := []string{"bomb_timeticking"}
	return newCSVRow, header, nil
}

type grenadeTracker struct {
	grenadeEvent events.GrenadeEvent
	grenadeTime  float64
}

type roundWinnerHandler struct {
	basicHandler        *basicHandler
	roundStartHandlerID dp.HandlerIdentifier
	roundEndHandlerID   dp.HandlerIdentifier
}

type roundWinnerManager struct {
	roundWinnerHandler *infoGenerationHandler
}

func (rh *roundWinnerHandler) Register() error {
	parser := *(rh.basicHandler.parser)
	rh.roundStartHandlerID = parser.RegisterEventHandler(rh.RoundStartHandler)
	rh.roundEndHandlerID = parser.RegisterEventHandler(rh.RoundEndHandler)
	return nil
}

func (rh *roundWinnerHandler) RoundStartHandler(e events.RoundStart) {

	rh.basicHandler.roundWinner = ""

}

func (rh *roundWinnerHandler) RoundEndHandler(e events.RoundEnd) {

	winTeam := e.Winner
	if winTeam == 2 {
		rh.basicHandler.roundWinner = "t"
	} else if winTeam == 3 {
		rh.basicHandler.roundWinner = "ct"
	} else {
		rh.basicHandler.roundWinner = "invalid"
	}
	if rh.basicHandler.roundDirPath != rh.basicHandler.rootPath {
		fileWrite, err := os.Create(rh.basicHandler.roundDirPath + "/winner.txt")
		checkError(err)
		defer fileWrite.Close()
		_, err = fileWrite.WriteString(rh.basicHandler.roundWinner)
		checkError(err)

	}

}

type matchData struct {
	matchIcons       [][][]utils.Icon
	matchTabularData [][][]string
	matchStatistics  [][][]string
}

func (md *matchData) CropData(index int) {
	md.matchIcons = md.matchIcons[:index]
	md.matchTabularData = md.matchTabularData[:index]
	md.matchStatistics = md.matchStatistics[:index]
}

func (md *matchData) AddNewRound() {
	md.matchIcons = append(md.matchIcons, [][]utils.Icon{})
	md.matchTabularData = append(md.matchTabularData, [][]string{})
	md.matchStatistics = append(md.matchStatistics, [][]string{})
}

type infoGenerationHandler struct {
	basicHandler                *basicHandler
	roundStartHandlerID         dp.HandlerIdentifier
	roundFreezetimeEndHandlerID dp.HandlerIdentifier
	frameDoneHandlerID          dp.HandlerIdentifier
	roundFreezeTime             bool
	generationIndex             int
	lastUpdate                  float64
	isNewRound                  bool
	updateInterval              float64
	roundEndRegistered          bool //set to true after processing the first frame subsequent to winner callout
	allIconGenerators           []IconGenerator
	allTabularGenerators        []TabularGenerator
	allStatGenerators           []StatGenerator
	matchData                   *matchData
	imgSize                     int
}

type infoGenerationManager struct {
	infoGenerationHandler *infoGenerationHandler
}

func (ih *infoGenerationHandler) RoundStartHandler(e events.RoundStart) {
	parser := *(ih.basicHandler.parser)
	gs := parser.GameState()
	ih.roundFreezeTime = true

	ih.basicHandler.roundNumber = gs.TeamCounterTerrorists().Score() + gs.TeamTerrorists().Score() + 1
	newScore := utils.PadLeft(strconv.Itoa(ih.basicHandler.roundNumber), "0", 2) + "_ct_" +
		utils.PadLeft(strconv.Itoa(gs.TeamCounterTerrorists().Score()), "0", 2) +
		"_t_" + utils.PadLeft(strconv.Itoa(gs.TeamTerrorists().Score()), "0", 2)

	ih.basicHandler.roundDirPath = ih.basicHandler.rootPath + "/" + newScore
	dirExists, _ := exists(ih.basicHandler.roundDirPath)
	ih.generationIndex = 0
	if !dirExists {
		err := os.MkdirAll(ih.basicHandler.roundDirPath, 0700)
		checkError(err)
	} else {
		RemoveContents(ih.basicHandler.roundDirPath)
	}
	ih.basicHandler.roundTabularPath = ih.basicHandler.roundDirPath + "/tabular.csv"
	ih.isNewRound = true
	roundCSV, err := os.Create(ih.basicHandler.roundTabularPath)
	checkError(err)
	defer roundCSV.Close()

	ih.basicHandler.roundStartTime = getCurrentTime(parser, ih.basicHandler.tickRate)
	ih.lastUpdate = 0.0

	if len(ih.matchData.matchIcons) > ih.basicHandler.roundNumber-1 { // match restart or round rollback
		ih.matchData.CropData(ih.basicHandler.roundNumber)
	} else if len(ih.matchData.matchIcons) < ih.basicHandler.roundNumber-1 {
		fmt.Println("missing match data")
	} else {
		ih.matchData.AddNewRound()
	}

}

func (ih *infoGenerationHandler) RoundFreezetimeEndHandler(e events.RoundFreezetimeEnd) {
	ih.roundFreezeTime = false
}

func (ih *infoGenerationHandler) FrameDoneHandler(e events.FrameDone) {

	if ih.isReadyForProcessing() {
		ih.processFrameEnd()
		if ih.basicHandler.roundWinner != "" {
			generateMap(ih.matchData.matchIcons[ih.basicHandler.roundNumber-1],
				ih.basicHandler.roundDirPath, ih.imgSize)
			writeToCSV(ih.matchData.matchTabularData[ih.basicHandler.roundNumber-1],
				ih.basicHandler.roundTabularPath)
			ih.roundEndRegistered = true
		}
	}
}

func (ih *infoGenerationHandler) isReadyForProcessing() bool {
	parser := *(ih.basicHandler.parser)
	gs := parser.GameState()
	currentRoundTime := getRoundTime(parser, ih.basicHandler.roundStartTime, ih.basicHandler.tickRate)
	roundDir := ih.basicHandler.roundDirPath
	if !(gs == nil) &&
		ih.basicHandler.roundDirPath != ih.basicHandler.rootPath &&
		!ih.roundFreezeTime && !ih.roundEndRegistered &&
		(currentRoundTime-ih.lastUpdate) > ih.updateInterval {

		return true
	}
	return false
}

func (ih *infoGenerationHandler) Register() error {
	parser := *(ih.basicHandler.parser)
	ih.roundStartHandlerID = parser.RegisterEventHandler(ih.RoundStartHandler)
	ih.roundFreezetimeEndHandlerID = parser.RegisterEventHandler(ih.RoundFreezetimeEndHandler)
	ih.frameDoneHandlerID = parser.RegisterEventHandler(ih.FrameDoneHandler)
	return nil
}

func (ih *infoGenerationHandler) processFrameEnd() {
	var tempIcons []utils.Icon
	var newIcons []utils.Icon
	for _, iconGenerator := range ih.allIconGenerators {
		tempIcons, err := iconGenerator.GetIcons()
		checkError(err)
		newIcons = append(newIcons, tempIcons...)
	}
	ih.matchData.matchIcons[ih.basicHandler.roundNumber-1] =
		append(ih.matchData.matchIcons[ih.basicHandler.roundNumber-1], newIcons)

	var newHeaderStat []string
	var newStat []string
	var newHeaderTabular []string
	var newTabular []string
	var tempHeader []string
	var tempData []string
	var err error
	for _, statGenerator := range ih.allStatGenerators {
		tempHeader, tempData, err = statGenerator.GetStatistics()
		checkError(err)
		newHeaderStat = append(newHeaderStat, tempHeader...)
		newStat = append(newStat, tempData...)
	}

	for _, tabularGenerator := range ih.allTabularGenerators {
		tempHeader, tempData, err = tabularGenerator.GetTabularData()
		checkError(err)
		newHeaderTabular = append(newHeaderTabular, tempHeader...)
		newTabular = append(newTabular, tempData...)
	}

	if ih.isNewRound {
		ih.matchData.matchStatistics[ih.basicHandler.roundNumber-1] =
			append(ih.matchData.matchStatistics[ih.basicHandler.roundNumber-1], newHeaderStat)

		ih.matchData.matchTabularData[ih.basicHandler.roundNumber-1] =
			append(ih.matchData.matchTabularData[ih.basicHandler.roundNumber-1], newHeaderTabular)
		ih.isNewRound = false
	}

	ih.matchData.matchStatistics[ih.basicHandler.roundNumber-1] =
		append(ih.matchData.matchStatistics[ih.basicHandler.roundNumber-1], newStat)
	ih.matchData.matchTabularData[ih.basicHandler.roundNumber-1] =
		append(ih.matchData.matchTabularData[ih.basicHandler.roundNumber-1], newTabular)

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

func ProcessDemoFile(demPath string, fileID int, destDir string, tickRate int) {
	f, err := os.Open(demPath)
	checkError(err)
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Erro no processamento do arquivo!", r)
		}
	}()
	p := dem.NewParser(f)
	defer f.Close()

	var currentState = ""
	var roundStartTime float64
	var lastUpdate = 0.0
	var updateInterval = 1.5
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

	imgSize := 800

	mapMetadata := metadata.MapNameToMap[header.MapName]
	var mapGenerator utils.MapGenerator
	var allIconGenerators []IconGenerator
	var allStatGenerators []StatGenerator
	var allTabularGenerators []TabularGenerator
	var basicHandler basicHandler

	mapGenerator.Setup(header.MapName)

	basicHandler.SetConfig(&p, tickRate, mapMetadata)

	var popHandler poppingGrenadeHandler
	popHandler.basicHandler = &basicHandler
	popHandler.SetBaseIcons()
	popHandler.Register()
	popManager := poppingGrenadeManager{popHandler: &popHandler}

	allIconGenerators = append(allIconGenerators, popManager)

	var bmbHandler bombHandler
	bmbHandler.basicHandler = &basicHandler
	bmbHandler.Register()
	bombManager := bombManager{bmbHandler: &bmbHandler}

	allTabularGenerators = append(allTabularGenerators, bombManager)
	allIconGenerators = append(allIconGenerators, bombManager)

	err = p.ParseToEnd()
	p.Close()

	checkError(err)

	// Parse to end
}

func main() {
	demPath := os.Args[2]
	destDir := os.Args[3]

	mode := flag.String("mode", "file", "process mode (file/dir)")
	fileID := 0
	tickRate, _ := strconv.Atoi(os.Args[4])
	flag.Parse()
	if *mode == "file" {
		ProcessDemoFile(demPath, fileID, destDir, tickRate)
	} else if *mode == "dir" {
		files, err := ioutil.ReadDir(demPath)
		if err != nil {
			log.Fatal(err)
		}
		for _, f := range files {
			ProcessDemoFile(demPath+"/"+f.Name(), fileID, destDir, tickRate)
			fileID++
		}

	} else {
		log.Fatal("invalid mode.")
	}

}

func writeToCSV(data [][]string, filePath string) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	checkError(err)
	writer := csv.NewWriter(file)

	err = writer.WriteAll(data)
	checkError(err)
}

func remakePlayerMappings(gs dem.GameState) map[int]*playerMapping {
	newAllPlayers := make(map[int]*playerMapping)
	players := gs.Participants().Playing()
	ctCount := 0
	tCount := 0
	playerIndex := 0
	for _, player := range players {
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
			playerIndex = 5 + ctCount
			ctCount++
		} else if isTR {
			playerIndex = tCount
			tCount++
		}
		newAllPlayers[player.UserID] = &playerMapping{playerObject: player}
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

func getRoundTime(p dem.Parser, roundStartTime float64, tickRate int) float64 {
	return getCurrentTime(p, tickRate) - roundStartTime
}

func getCurrentTime(p dem.Parser, tickRate int) float64 {
	currentFrame := p.CurrentFrame()
	return float64(currentFrame) / float64(tickRate)
}

func generateMap(mapGenerator *utils.MapGenerator, iconLists [][]Icon, roundPath string, imgSize int) {
	roundMaps := mapGenerator.DrawMap(iconsLists)
	imageIndex := 0
	for _, imgOriginal := range roundMaps {
		img := resize.Resize(uint(imgSize), 0, imgOriginal, resize.Bilinear)
		third, err := os.Create(roundPath + "/output_map" +
			utils.PadLeft(strconv.Itoa(*imageIndex), "0", 2) + ".jpg")
		if err != nil {
			log.Fatalf("failed to create: %s", err)
		}
		err = jpeg.Encode(third, img, &jpeg.Options{jpeg.DefaultQuality})
		checkError(err)
		imageIndex++
		third.Close()

	}

}
