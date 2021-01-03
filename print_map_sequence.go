package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"image/jpeg"
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

//struct to gather player related data
type playerMapping struct {
	playerObject *common.Player
}

//generic event handler registering interface
type CompositeEventHandler interface {
	Register(*basicHandler) error
	//Unregister() error
}

//IconGenerators generate icons on output map
type IconGenerator interface {
	GetIcons() ([]utils.Icon, error)
}

//TabularGenerators generate data rows on output file
type TabularGenerator interface {
	GetTabularData() ([]string, []string, error) //header, data, error
}

//StatGenerators generate rows on output stat file
type StatGenerator interface {
	GetStatistics() ([]string, []string, error) //header, data, error
}

//Interface to RoundStart event subscribers
type RoundStartSubscriber interface {
	RoundStartHandler(events.RoundStart)
}

//Interface to RoundEnd event subscribers
type RoundEndSubscriber interface {
	RoundEndHandler(events.RoundEnd)
}

//Interface to GrenadeEventIf event subscribers
type GrenadeEventIfSubscriber interface {
	CompositeEventHandler
	GrenadeEventIfHandler(events.GrenadeEventIf)
}

//Interface to RoundFreezetimeEnd event subscribers
type RoundFreezetimeEndSubscriber interface {
	RoundFreezetimeEndHandler(events.RoundFreezetimeEnd)
}

//Interface to BombPlanted event subscribers
type BombPlantedSubscriber interface {
	BombPlantedHandler(events.BombPlanted)
}

//Interface to FrameDone event subscribers
type FrameDoneSubscriber interface {
	FrameDoneHandler(events.FrameDone)
}

//basic shared handler for parsing
type basicHandler struct {
	parser      *dem.Parser
	tickRate    int
	mapMetadata metadata.Map

	roundStartHandlerID   dp.HandlerIdentifier
	roundStartSubscribers []RoundStartSubscriber

	roundEndHandlerID   dp.HandlerIdentifier
	roundEndSubscribers []RoundEndSubscriber

	grenadeEventIfHandlerID   dp.HandlerIdentifier
	grenadeEventIfSubscribers []GrenadeEventIfSubscriber

	roundFreezeTimeEndHandlerID   dp.HandlerIdentifier
	roundFreezeTimeEndSubscribers []RoundFreezetimeEndSubscriber

	bombPlantedHandlerID   dp.HandlerIdentifier
	bombPlantedSubscribers []BombPlantedSubscriber

	frameDoneHandlerID   dp.HandlerIdentifier
	frameDoneSubscribers []FrameDoneSubscriber

	roundStartTime   float64
	rootMatchPath    string
	roundDirPath     string
	roundWinner      string
	roundTabularPath string
	roundStatPath    string
	currentTime      float64
	roundNumber      int
	frameGroup       int
	isMatchStarted   bool
	roundFreezeTime  bool
}

func (bh *basicHandler) RegisterBasicEvents() error {
	parser := *(bh.parser)
	bh.roundStartHandlerID = parser.RegisterEventHandler(bh.RoundStartHandler)
	bh.roundEndHandlerID = parser.RegisterEventHandler(bh.RoundEndHandler)
	bh.roundFreezeTimeEndHandlerID = parser.RegisterEventHandler(bh.RoundFreezetimeEndHandler)
	return nil
}

func (bh *basicHandler) Setup(parser *dem.Parser, tickRate int, mapMetadata metadata.Map, rootMatchPath string) error {
	bh.parser = parser
	bh.tickRate = tickRate
	bh.mapMetadata = mapMetadata
	bh.rootMatchPath = rootMatchPath
	bh.roundDirPath = rootMatchPath
	return nil
}

func (bh *basicHandler) UpdateTime() {
	bh.currentTime = getCurrentTime(*(bh.parser), bh.tickRate)
}

func (bh *basicHandler) GetTabularData() ([]string, []string, error) {
	parser := *(bh.parser)
	newCSVRow := []string{"0"}
	currentRoundTime := getCurrentTime(parser, bh.tickRate)

	newCSVRow[0] = strconv.FormatFloat(currentRoundTime-bh.roundStartTime, 'f', -1, 32)
	header := []string{"round_time"}
	return header, newCSVRow, nil

}

func (bh *basicHandler) RegisterRoundStartSubscriber(rs RoundStartSubscriber) {
	parser := *(bh.parser)
	if bh.roundStartHandlerID == nil {
		bh.roundStartHandlerID = parser.RegisterEventHandler(bh.RoundStartHandler)
	}

	bh.roundStartSubscribers = append(bh.roundStartSubscribers, rs)

}

func (bh *basicHandler) RoundStartHandler(e events.RoundStart) {
	bh.UpdateTime()
	parser := *(bh.parser)
	gs := parser.GameState()

	bh.roundFreezeTime = true
	bh.roundWinner = ""
	bh.frameGroup = 0
	bh.isMatchStarted = true
	bh.roundNumber = gs.TeamCounterTerrorists().Score() + gs.TeamTerrorists().Score() + 1
	newScore := utils.PadLeft(strconv.Itoa(bh.roundNumber), "0", 2) + "_ct_" +
		utils.PadLeft(strconv.Itoa(gs.TeamCounterTerrorists().Score()), "0", 2) +
		"_t_" + utils.PadLeft(strconv.Itoa(gs.TeamTerrorists().Score()), "0", 2)

	bh.roundDirPath = bh.rootMatchPath + "/" + newScore
	bh.roundStartTime = bh.currentTime

	for _, subscriber := range bh.roundStartSubscribers {
		subscriber.RoundStartHandler(e)
	}

}

func (bh *basicHandler) RegisterRoundEndSubscriber(rs RoundEndSubscriber) {
	parser := *(bh.parser)
	if bh.roundEndHandlerID == nil {
		bh.roundEndHandlerID = parser.RegisterEventHandler(bh.RoundEndHandler)
	}

	bh.roundEndSubscribers = append(bh.roundEndSubscribers, rs)

}

func (bh *basicHandler) RoundEndHandler(e events.RoundEnd) {
	bh.UpdateTime()
	winTeam := e.Winner
	if winTeam == 2 {
		bh.roundWinner = "t"
	} else if winTeam == 3 {
		bh.roundWinner = "ct"
	} else {
		bh.roundWinner = "invalid"
	}
	if bh.roundDirPath != bh.rootMatchPath {
		fileWrite, err := os.Create(bh.roundDirPath + "/winner.txt")
		checkError(err)
		defer fileWrite.Close()
		_, err = fileWrite.WriteString(bh.roundWinner)
		checkError(err)

	}

	for _, subscriber := range bh.roundEndSubscribers {
		subscriber.RoundEndHandler(e)
	}

}

func (bh *basicHandler) RegisterGrenadeEventIfSubscriber(rs GrenadeEventIfSubscriber) {
	parser := *(bh.parser)
	if bh.grenadeEventIfHandlerID == nil {
		bh.grenadeEventIfHandlerID = parser.RegisterEventHandler(bh.GrenadeEventIfHandler)
	}

	bh.grenadeEventIfSubscribers = append(bh.grenadeEventIfSubscribers, rs)

}

func (bh *basicHandler) GrenadeEventIfHandler(e events.GrenadeEventIf) {
	bh.UpdateTime()
	for _, subscriber := range bh.grenadeEventIfSubscribers {
		subscriber.GrenadeEventIfHandler(e)
	}
}

func (bh *basicHandler) RegisterRoundFreezetimeEndSubscriber(rs RoundFreezetimeEndSubscriber) {
	parser := *(bh.parser)
	if bh.roundFreezeTimeEndHandlerID == nil {
		bh.roundFreezeTimeEndHandlerID = parser.RegisterEventHandler(bh.RoundFreezetimeEndHandler)
	}

	bh.roundFreezeTimeEndSubscribers = append(bh.roundFreezeTimeEndSubscribers, rs)

}

func (bh *basicHandler) RoundFreezetimeEndHandler(e events.RoundFreezetimeEnd) {
	bh.UpdateTime()
	bh.roundFreezeTime = false

	for _, subscriber := range bh.roundFreezeTimeEndSubscribers {
		subscriber.RoundFreezetimeEndHandler(e)
	}
}

func (bh *basicHandler) RegisterBombPlantedSubscriber(rs BombPlantedSubscriber) {
	parser := *(bh.parser)
	if bh.bombPlantedHandlerID == nil {
		bh.bombPlantedHandlerID = parser.RegisterEventHandler(bh.BombPlantedHandler)
	}

	bh.bombPlantedSubscribers = append(bh.bombPlantedSubscribers, rs)

}

func (bh *basicHandler) BombPlantedHandler(e events.BombPlanted) {
	bh.UpdateTime()

	for _, subscriber := range bh.bombPlantedSubscribers {
		subscriber.BombPlantedHandler(e)
	}
}

func (bh *basicHandler) RegisterFrameDoneSubscriber(rs FrameDoneSubscriber) {
	parser := *(bh.parser)
	if bh.frameDoneHandlerID == nil {
		bh.frameDoneHandlerID = parser.RegisterEventHandler(bh.FrameDoneHandler)
	}

	bh.frameDoneSubscribers = append(bh.frameDoneSubscribers, rs)

}

func (bh *basicHandler) FrameDoneHandler(e events.FrameDone) {
	bh.UpdateTime()

	for _, subscriber := range bh.frameDoneSubscribers {
		subscriber.FrameDoneHandler(e)
	}
}

type poppingGrenadeHandler struct {
	basicHandler   *basicHandler
	activeGrenades []*grenadeTracker
	baseIcons      map[common.EquipmentType]utils.Icon
}

func (ph *poppingGrenadeHandler) Register(bh *basicHandler) error {
	ph.basicHandler = bh
	bh.RegisterGrenadeEventIfSubscriber(interface{}(ph).(GrenadeEventIfSubscriber))
	bh.RegisterRoundStartSubscriber(interface{}(ph).(RoundStartSubscriber))
	return nil
}

func (ph *poppingGrenadeHandler) RoundStartHandler(e events.RoundStart) {
	ph.activeGrenades = nil
}

//e holds smoke start/expired or inferno start/expired and other grenade events
func (ph *poppingGrenadeHandler) GrenadeEventIfHandler(e events.GrenadeEventIf) {

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

// func (ph *poppingGrenadeHandler) Unregister() error {
// 	parser := *(ph.basicHandler.parser)
// 	parser.UnregisterEventHandler(ph.grenadeStartHandlerID)
// 	return nil

// }

func (ph *poppingGrenadeHandler) GetIcons() ([]utils.Icon, error) {
	var iconList []utils.Icon
	for _, activeGrenade := range ph.activeGrenades {
		newIcon := ph.baseIcons[activeGrenade.grenadeEvent.GrenadeType]
		x, y := ph.basicHandler.mapMetadata.TranslateScale(activeGrenade.grenadeEvent.Position.X, activeGrenade.grenadeEvent.Position.Y)
		newIcon.X, newIcon.Y = x, y
		iconList = append(iconList, newIcon)
	}
	return iconList, nil
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

type playerInfoHandler struct {
	basicHandler   *basicHandler
	playerMappings map[int]*playerMapping
	sortedUserIDs  []int
}

func (ph *playerInfoHandler) Register(bh *basicHandler) error {
	ph.basicHandler = bh
	bh.RegisterRoundStartSubscriber(interface{}(ph).(RoundStartSubscriber))
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

					icon = "ct_"
					if player.HasDefuseKit() {
						newIcon := utils.Icon{X: x, Y: y, IconName: "kit"} //t or ct icon
						iconList = append(iconList, newIcon)
					}
					ctCount++
					playerCount = ctCount
				} else if isTR {
					icon = "terrorist_"

					tCount++
					playerCount = tCount
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

func (ph *playerInfoHandler) GetTabularData() (newHeader []string, newCSVRow []string, err error) {

	tempCSV, tempHeader := ph.getPlayersHPAndFlashtime()
	newCSVRow = append(newCSVRow, tempCSV...)
	newHeader = append(newHeader, tempHeader...)

	tempCSV, tempHeader = ph.processPlayerWeapons()
	newCSVRow = append(newCSVRow, tempCSV...)
	newHeader = append(newHeader, tempHeader...)
	return newHeader, newCSVRow, err
}

func (ph *playerInfoHandler) GetIcons() ([]utils.Icon, error) {

	return ph.processPlayerPositions(), nil

}

type bombHandler struct {
	basicHandler    *basicHandler
	bombPlanted     bool
	bombPlantedTime float64
	baseIcons       map[string]utils.Icon
}

func (bmbh *bombHandler) Register(bh *basicHandler) error {
	bmbh.basicHandler = bh
	bh.RegisterBombPlantedSubscriber(interface{}(bmbh).(BombPlantedSubscriber))
	bh.RegisterRoundStartSubscriber(interface{}(bmbh).(RoundStartSubscriber))
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

func (bh *bombHandler) GetIcons() (icons []utils.Icon, err error) {
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

func (bh *bombHandler) GetTabularData() ([]string, []string, error) {
	newCSVRow := []string{"0"}
	if bh.bombPlanted {
		newCSVRow[0] = strconv.FormatFloat(bh.basicHandler.currentTime-bh.bombPlantedTime, 'f', -1, 32)
	}

	header := []string{"bomb_timeticking"}
	return header, newCSVRow, nil
}

type grenadeTracker struct {
	grenadeEvent events.GrenadeEvent
	grenadeTime  float64
}

type matchData struct {
	matchIcons       [][][]utils.Icon //dimensions: rounds x frames x icons
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

func (md *matchData) AddNewFrameGroup(roundNumber int) {
	if len(md.matchIcons[roundNumber]) == 0 {
		md.matchTabularData[roundNumber] = append(md.matchTabularData[roundNumber], []string{})
		md.matchStatistics[roundNumber] = append(md.matchStatistics[roundNumber], []string{})
	}
	md.matchIcons[roundNumber] = append(md.matchIcons[roundNumber], []utils.Icon{})
	md.matchTabularData[roundNumber] = append(md.matchTabularData[roundNumber], []string{})
	md.matchStatistics[roundNumber] = append(md.matchStatistics[roundNumber], []string{})
}

type infoGenerationHandler struct {
	basicHandler         *basicHandler
	frameDoneHandlerID   dp.HandlerIdentifier
	generationIndex      int
	lastUpdate           float64
	isNewRound           bool
	updateInterval       float64
	roundEndRegistered   bool //set to true after processing the first frame subsequent to winner callout
	allIconGenerators    *[]IconGenerator
	allTabularGenerators *[]TabularGenerator
	allStatGenerators    *[]StatGenerator
	mapGenerator         utils.MapGenerator
	matchData            *matchData
	imgSize              int
}

func (ih *infoGenerationHandler) Register(bh *basicHandler) error {
	ih.basicHandler = bh
	bh.RegisterRoundStartSubscriber(interface{}(ih).(RoundStartSubscriber))
	bh.RegisterFrameDoneSubscriber(interface{}(ih).(FrameDoneSubscriber))
	return nil
}

// func (ih *infoGenerationHandler) Unregister(bh *basicHandler) error {
// 	ih.basicHandler = bh
// 	bh.RegisterCompositeEventHandler(interface{}(ih).(CompositeEventHandler))
// 	return nil
// }

func (ih *infoGenerationHandler) RoundStartHandler(e events.RoundStart) {

	dirExists, _ := exists(ih.basicHandler.roundDirPath)
	ih.generationIndex = 0
	ih.roundEndRegistered = false
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

	ih.lastUpdate = 0.0

	if len(ih.matchData.matchIcons) > ih.basicHandler.roundNumber-1 { // match restart or round rollback
		ih.matchData.CropData(ih.basicHandler.roundNumber - 1)
		ih.matchData.AddNewRound()
	} else if len(ih.matchData.matchIcons) < ih.basicHandler.roundNumber-1 {
		fmt.Println("missing match data")
	} else {
		ih.matchData.AddNewRound()
	}

}

func (ih *infoGenerationHandler) FrameDoneHandler(e events.FrameDone) {

	if ih.isReadyForProcessing() {
		ih.processFrameEnd()
		if ih.basicHandler.roundWinner != "" {
			generateRoundMaps(ih.mapGenerator, ih.matchData.matchIcons[ih.basicHandler.roundNumber-1],
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
	if !(gs == nil) &&
		ih.basicHandler.isMatchStarted &&
		!ih.basicHandler.roundFreezeTime && !ih.roundEndRegistered &&
		(currentRoundTime-ih.lastUpdate) > ih.updateInterval {

		return true
	}
	return false
}

func (ih *infoGenerationHandler) Setup(imgSize int, updateInterval float64, allIconGenerators *[]IconGenerator, allTabularGenerators *[]TabularGenerator,
	allStatGenerators *[]StatGenerator) error {

	var mapGenerator utils.MapGenerator
	mapGenerator.Setup(ih.basicHandler.mapMetadata.Name, imgSize)
	ih.mapGenerator = mapGenerator
	ih.updateInterval = updateInterval
	ih.matchData = new(matchData)

	ih.allIconGenerators = allIconGenerators
	ih.allTabularGenerators = allTabularGenerators
	ih.allStatGenerators = allStatGenerators

	return nil
}

func (ih *infoGenerationHandler) processFrameEnd() {
	var newIcons []utils.Icon
	for _, iconGenerator := range *ih.allIconGenerators {
		tempIcons, err := iconGenerator.GetIcons()
		checkError(err)
		newIcons = append(newIcons, tempIcons...)
	}
	ih.matchData.AddNewFrameGroup(ih.basicHandler.roundNumber - 1)
	ih.matchData.matchIcons[ih.basicHandler.roundNumber-1][ih.basicHandler.frameGroup] =
		append(ih.matchData.matchIcons[ih.basicHandler.roundNumber-1][ih.basicHandler.frameGroup], newIcons...)

	var newHeaderStat []string
	var newStat []string
	var newHeaderTabular []string
	var newTabular []string
	var tempHeader []string
	var tempData []string
	var err error
	for _, statGenerator := range *ih.allStatGenerators {
		tempHeader, tempData, err = statGenerator.GetStatistics()
		checkError(err)
		newHeaderStat = append(newHeaderStat, tempHeader...)
		newStat = append(newStat, tempData...)
	}

	for _, tabularGenerator := range *ih.allTabularGenerators {
		tempHeader, tempData, err = tabularGenerator.GetTabularData()
		checkError(err)
		newHeaderTabular = append(newHeaderTabular, tempHeader...)
		newTabular = append(newTabular, tempData...)
	}

	if ih.isNewRound {
		ih.matchData.matchStatistics[ih.basicHandler.roundNumber-1][ih.basicHandler.frameGroup] =
			append(ih.matchData.matchStatistics[ih.basicHandler.roundNumber-1][ih.basicHandler.frameGroup], newHeaderStat...)

		ih.matchData.matchTabularData[ih.basicHandler.roundNumber-1][ih.basicHandler.frameGroup] =
			append(ih.matchData.matchTabularData[ih.basicHandler.roundNumber-1][ih.basicHandler.frameGroup], newHeaderTabular...)
		ih.isNewRound = false
	}

	ih.matchData.matchStatistics[ih.basicHandler.roundNumber-1][ih.basicHandler.frameGroup+1] =
		append(ih.matchData.matchStatistics[ih.basicHandler.roundNumber-1][ih.basicHandler.frameGroup+1], newStat...)
	ih.matchData.matchTabularData[ih.basicHandler.roundNumber-1][ih.basicHandler.frameGroup+1] =
		append(ih.matchData.matchTabularData[ih.basicHandler.roundNumber-1][ih.basicHandler.frameGroup+1], newTabular...)

	ih.basicHandler.frameGroup = ih.basicHandler.frameGroup + 1
	parser := *(ih.basicHandler.parser)
	ih.lastUpdate = getRoundTime(parser, ih.basicHandler.roundStartTime, ih.basicHandler.tickRate)
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

func processDemoFile(demPath string, fileID int, destDir string, tickRate int) {
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
	rootMatchPath := destDir + "/" + header.MapName + "/" + strconv.Itoa(fileID)
	dirExists, _ := exists(rootMatchPath)
	if !dirExists {
		err = os.MkdirAll(rootMatchPath, 0700)
		checkError(err)
	}

	imgSize := 800

	mapMetadata := metadata.MapNameToMap[header.MapName]
	var mapGenerator utils.MapGenerator
	var allIconGenerators []IconGenerator
	var allStatGenerators []StatGenerator
	var allTabularGenerators []TabularGenerator
	var basicHandler basicHandler

	mapGenerator.Setup(header.MapName, imgSize)

	basicHandler.Setup(&p, tickRate, mapMetadata, rootMatchPath)
	basicHandler.RegisterBasicEvents()
	allTabularGenerators = append(allTabularGenerators, &basicHandler)

	var popHandler poppingGrenadeHandler
	popHandler.SetBaseIcons()
	popHandler.Register(&basicHandler)
	allIconGenerators = append(allIconGenerators, &popHandler)

	var bmbHandler bombHandler
	bmbHandler.Register(&basicHandler)
	allTabularGenerators = append(allTabularGenerators, &bmbHandler)
	allIconGenerators = append(allIconGenerators, &bmbHandler)

	var playerHandler playerInfoHandler
	playerHandler.Register(&basicHandler)
	allTabularGenerators = append(allTabularGenerators, &playerHandler)
	allIconGenerators = append(allIconGenerators, &playerHandler)

	var infoHandler infoGenerationHandler
	updateInterval := 2.0 //2s between framegroups
	infoHandler.Register(&basicHandler)
	infoHandler.Setup(imgSize, updateInterval, &allIconGenerators, &allTabularGenerators, &allStatGenerators)

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
		processDemoFile(demPath, fileID, destDir, tickRate)
	} else if *mode == "dir" {

		filepath.Walk(demPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
				return err
			}
			if info.IsDir() {
				return nil
			}
			processDemoFile(path, fileID, destDir, tickRate)
			fileID++
			return nil
		})

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

			ctCount++
		} else if isTR {

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

func getRoundTime(p dem.Parser, roundStartTime float64, tickRate int) float64 {
	return getCurrentTime(p, tickRate) - roundStartTime
}

func getCurrentTime(p dem.Parser, tickRate int) float64 {
	currentFrame := p.CurrentFrame()
	return float64(currentFrame) / float64(tickRate)
}

func generateRoundMaps(mapGenerator utils.MapGenerator, iconLists [][]utils.Icon, roundPath string, imgSize int) {
	roundMaps := mapGenerator.DrawMap(iconLists)
	for imageIndex, imgOriginal := range roundMaps {
		img := resize.Resize(uint(imgSize), 0, imgOriginal, resize.Bilinear)
		third, err := os.Create(roundPath + "/output_map" +
			utils.PadLeft(strconv.Itoa(imageIndex), "0", 2) + ".jpg")
		if err != nil {
			log.Fatalf("failed to create: %s", err)
		}
		err = jpeg.Encode(third, img, &jpeg.Options{jpeg.DefaultQuality})
		checkError(err)
		imageIndex++
		third.Close()

	}

}
