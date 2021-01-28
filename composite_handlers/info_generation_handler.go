package composite_handlers

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
	dp "github.com/markus-wa/godispatch"
	database "github.com/mrdbarros/csgo_analyze/database"
	map_builder "github.com/mrdbarros/csgo_analyze/map_builder"
	utils "github.com/mrdbarros/csgo_analyze/utils"
)

type matchData struct {
	matchIcons                      [][][]map_builder.Icon //dimensions: rounds x frames x icons
	matchPeriodicTabularDataHeaders []string
	matchPeriodicTabularData        [][][]float64
	matchStatisticsHeaders          []string
	matchStatistics                 [][]float64
}

func (md *matchData) CropData(index int) {
	md.matchIcons = md.matchIcons[:index]
	md.matchPeriodicTabularData = md.matchPeriodicTabularData[:index]
	md.matchStatistics = md.matchStatistics[:index]
}

func (md *matchData) AddNewRound() {
	md.matchIcons = append(md.matchIcons, [][]map_builder.Icon{})
	md.matchPeriodicTabularData = append(md.matchPeriodicTabularData, [][]float64{})
	md.matchStatistics = append(md.matchStatistics, []float64{})
}

func (md *matchData) AddNewFrameGroup(roundNumber int) {
	if len(md.matchIcons[roundNumber]) == 0 {
		md.matchPeriodicTabularData[roundNumber] = append(md.matchPeriodicTabularData[roundNumber], []float64{})
	}
	md.matchIcons[roundNumber] = append(md.matchIcons[roundNumber], []map_builder.Icon{})
	md.matchPeriodicTabularData[roundNumber] = append(md.matchPeriodicTabularData[roundNumber], []float64{})

}

type InfoGenerationHandler struct {
	basicHandler            *BasicHandler
	frameDoneHandlerID      dp.HandlerIdentifier
	generationIndex         int
	lastUpdate              float64
	isNewRound              bool
	updateInterval          float64
	roundEndRegistered      bool //set to true after generating roundendofficial info
	matchEndRegisted        bool
	allIconGenerators       *[]PeriodicIconGenerator
	allTabularGenerators    *[]PeriodicTabularGenerator
	allStatGenerators       *[]StatGenerator
	allPlayerStatCalculator *[]PlayerStatisticCalculator
	mapGenerator            map_builder.MapGenerator
	matchData               *matchData
	imgSize                 int
	demFileHash             string

	rootMatchPath    string
	roundDirPath     string
	roundTabularPath string
	roundStatPath    string
	playerStatPath   string
}

func (ih *InfoGenerationHandler) Register(bh *BasicHandler) error {
	ih.basicHandler = bh

	bh.RegisterRoundStartSubscriber(interface{}(ih).(RoundStartSubscriber))
	bh.RegisterFrameDoneSubscriber(interface{}(ih).(FrameDoneSubscriber))
	bh.RegisterRoundEndOfficialSubscriber(interface{}(ih).(RoundEndOfficialSubscriber))
	bh.RegisterRoundEndSubscriber(interface{}(ih).(RoundEndSubscriber))

	return nil
}

func (ih *InfoGenerationHandler) RoundStartHandler(e events.RoundStart) {
	if !ih.basicHandler.isMatchEnded {
		ih.roundDirPath = ih.rootMatchPath + "/" + ih.basicHandler.currentScore
		dirExists, _ := utils.Exists(ih.roundDirPath)
		ih.generationIndex = 0
		ih.roundEndRegistered = false

		if !dirExists {
			err := os.MkdirAll(ih.roundDirPath, 0700)
			utils.CheckError(err)
		} else {
			utils.RemoveContents(ih.roundDirPath)
		}

		ih.roundTabularPath = ih.roundDirPath + "/periodic_data.csv"
		ih.roundStatPath = ih.roundDirPath + "/statistics.csv"
		ih.playerStatPath = ih.roundDirPath + "/player_statistics.csv"
		ih.isNewRound = true
		roundPeriodicDataCSV, err := os.Create(ih.roundTabularPath)
		utils.CheckError(err)
		defer roundPeriodicDataCSV.Close()

		roundStatCSV, err := os.Create(ih.roundStatPath)
		utils.CheckError(err)
		defer roundStatCSV.Close()

		playerStatCSV, err := os.Create(ih.playerStatPath)
		utils.CheckError(err)
		defer playerStatCSV.Close()

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

}

func (ih *InfoGenerationHandler) GetFullRoundStatistics() (data [][]string) {
	var tempHeader []string
	var tempData []float64
	var err error
	var stringData []string
	var framedData [10][]string
	firstPlayer := true
	data = append(data, []string{"Name"})
	for _, playerMapping := range ih.basicHandler.playerMappings[ih.basicHandler.roundNumber-1] {
		player := playerMapping.playerObject

		for j, playerStatCalculator := range *ih.allPlayerStatCalculator {
			tempHeader, tempData, err = playerStatCalculator.GetRoundStatistic(ih.basicHandler.roundNumber, player.SteamID64)

			utils.CheckError(err)

			if j == 0 {

				stringData = append([]string{playerMapping.playerObject.Name}, utils.FloatSliceToString(tempData)...)
			} else {
				stringData = append(stringData, utils.FloatSliceToString(tempData)...)
			}

			if firstPlayer {
				data[0] = append(data[0], tempHeader...)
			}

		}
		framedData[playerMapping.currentSlot] = append(framedData[playerMapping.currentSlot], stringData...)
		firstPlayer = false

	}
	data = append(data, framedData[:]...)
	return data

}

func (ih *InfoGenerationHandler) processRoundEnd() {
	if ih.basicHandler.roundWinner != "" && !ih.matchEndRegisted {
		fmt.Println("Generating round ", ih.basicHandler.roundNumber)

		var tempHeader []string
		var tempData []float64
		var newHeaderStat []string
		var newStat []float64
		var err error

		if ih.roundDirPath != ih.rootMatchPath {
			fileWrite, err := os.Create(ih.roundDirPath + "/winner.txt")
			utils.CheckError(err)
			defer fileWrite.Close()
			_, err = fileWrite.WriteString(ih.basicHandler.roundWinner)
			utils.CheckError(err)

		}

		for _, statGenerator := range *ih.allStatGenerators {
			newHeaderStat, newStat, err = statGenerator.GetStatistics()
			utils.CheckError(err)
			newHeaderStat = append(newHeaderStat, tempHeader...)
			newStat = append(newStat, tempData...)

		}
		generalStatistics := append([][]string{newHeaderStat}, utils.FloatSliceToString(newStat))

		playerStatistics := ih.GetFullRoundStatistics()

		if ih.basicHandler.roundNumber == 1 && len(*ih.allStatGenerators) > 0 {
			ih.matchData.matchStatisticsHeaders = nil
			ih.matchData.matchStatisticsHeaders = append(ih.matchData.matchStatisticsHeaders,
				newHeaderStat...)

		}
		if len(*ih.allStatGenerators) > 0 {
			ih.matchData.matchStatistics[ih.basicHandler.roundNumber-1] = append(ih.matchData.matchStatistics[ih.basicHandler.roundNumber-1],
				newStat...)
		}

		allTabularData := append([][]string{ih.matchData.matchPeriodicTabularDataHeaders}, utils.FloatMatrixToString(ih.matchData.matchPeriodicTabularData[ih.basicHandler.roundNumber-1])...)

		map_builder.GenerateRoundMaps(ih.mapGenerator, ih.matchData.matchIcons[ih.basicHandler.roundNumber-1],
			ih.roundDirPath, ih.imgSize)
		utils.WriteToCSV(allTabularData, ih.roundTabularPath)
		utils.WriteToCSV(generalStatistics, ih.roundStatPath)
		utils.WriteToCSV(playerStatistics, ih.playerStatPath)

		ih.checkAndGenerateMatchEndStatistics()
		ih.roundEndRegistered = true
	}
}

func (ih *InfoGenerationHandler) RoundEndOfficialHandler(e events.RoundEndOfficial) {
	if !ih.basicHandler.isMatchEnded {
		ih.processRoundEnd()
	}

}

func (ih *InfoGenerationHandler) RoundEndHandler(e events.RoundEnd) {
	if ih.basicHandler.isMatchEnded {
		ih.processRoundEnd()
	}

}

func (ih *InfoGenerationHandler) checkAndGenerateMatchEndStatistics() {

	if ih.basicHandler.isMatchEnded {
		fmt.Println("Generating match statistics")
		if ih.roundDirPath != ih.rootMatchPath {
			data := ih.GetFullMatchStatistics()
			fileWrite, err := os.Create(ih.rootMatchPath + "/match_statistics.csv")
			utils.CheckError(err)
			writer := csv.NewWriter(fileWrite)

			err = writer.WriteAll(data)
			utils.CheckError(err)
			defer fileWrite.Close()
			ih.matchEndRegisted = true
		}
	}

}

func (ih *InfoGenerationHandler) GetFullMatchStatistics() (data [][]string) {
	var tempHeader []string
	var tempData []float64
	var err error
	var stringData []string
	var framedData [][]string
	var statsIDs [][]int

	firstPlayer := true
	data = append(data, []string{"Name", "SteamID"})
	dbConn := database.OpenDBConn()

	matchID := database.InsertMatch(dbConn, ih.basicHandler.fileName, ih.demFileHash, ih.basicHandler.mapMetadata.Name,
		ih.basicHandler.terroristFirstTeamscore, ih.basicHandler.ctFirstTeamScore, ih.basicHandler.matchDatetime, true)

	var allPlayers map[uint64]playerMapping
	allPlayers = make(map[uint64]playerMapping)
	for roundID, _ := range ih.basicHandler.playerMappings {
		for _, playerMapping := range ih.basicHandler.playerMappings[roundID] {
			if _, ok := allPlayers[playerMapping.playerObject.SteamID64]; !ok {
				allPlayers[playerMapping.playerObject.SteamID64] = playerMapping
			}
		}
	}

	for _, playerMapping := range allPlayers {
		player := playerMapping.playerObject

		for j, playerStatCalculator := range *ih.allPlayerStatCalculator {
			tempHeader, tempData, err = playerStatCalculator.GetMatchStatistic(player.SteamID64)
			utils.CheckError(err)

			if j == 0 {
				database.InsertPlayer(dbConn, playerMapping.playerObject.SteamID64, playerMapping.playerObject.Name)
				stringData = append([]string{playerMapping.playerObject.Name,
					strconv.FormatUint(playerMapping.playerObject.SteamID64, 10)}, utils.FloatSliceToString(tempData)...)
			} else {
				stringData = append(stringData, utils.FloatSliceToString(tempData)...)
			}

			if firstPlayer {
				data[0] = append(data[0], tempHeader...)
				statsIDs = append(statsIDs, database.InsertBaseStatistics(dbConn, tempHeader))
			}
			database.InsertStatisticsFacts(dbConn, statsIDs[j], tempData, player.SteamID64, matchID)

		}

		framedData = append(framedData, stringData)
		firstPlayer = false

	}
	data = append(data, framedData...)
	return data

}

func (ih *InfoGenerationHandler) FrameDoneHandler(e events.FrameDone) {

	if ih.isReadyForProcessing() {
		ih.processFrameEnd()
	}
}

func (ih *InfoGenerationHandler) isReadyForProcessing() bool {
	parser := *(ih.basicHandler.parser)
	gs := parser.GameState()
	currentRoundTime := utils.GetRoundTime(parser, ih.basicHandler.roundStartTime, ih.basicHandler.tickRate)
	if !(gs == nil) &&
		ih.basicHandler.isMatchStarted &&
		!ih.basicHandler.roundFreezeTime && !ih.roundEndRegistered &&
		(currentRoundTime-ih.lastUpdate) > ih.updateInterval {

		return true
	}
	return false
}

func (ih *InfoGenerationHandler) Setup(imgSize int, updateInterval float64, rootMatchPath string, demFileHash string,
	allIconGenerators *[]PeriodicIconGenerator, allTabularGenerators *[]PeriodicTabularGenerator,
	allStatGenerators *[]StatGenerator, allPlayerStatCalculators *[]PlayerStatisticCalculator) error {

	var mapGenerator map_builder.MapGenerator
	mapGenerator.Setup(ih.basicHandler.mapMetadata.Name, imgSize)
	ih.mapGenerator = mapGenerator
	ih.updateInterval = updateInterval
	ih.matchData = new(matchData)
	ih.rootMatchPath = rootMatchPath
	ih.roundDirPath = rootMatchPath
	ih.demFileHash = demFileHash

	ih.allIconGenerators = allIconGenerators
	ih.allTabularGenerators = allTabularGenerators
	ih.allStatGenerators = allStatGenerators
	ih.allPlayerStatCalculator = allPlayerStatCalculators

	return nil
}

func (ih *InfoGenerationHandler) processFrameEnd() {
	var newIcons []map_builder.Icon

	for _, iconGenerator := range *ih.allIconGenerators {
		iconGenerator.Update()
		tempIcons, err := iconGenerator.GetPeriodicIcons()
		utils.CheckError(err)
		newIcons = append(newIcons, tempIcons...)
	}
	ih.matchData.AddNewFrameGroup(ih.basicHandler.roundNumber - 1)
	ih.matchData.matchIcons[ih.basicHandler.roundNumber-1][ih.basicHandler.frameGroup] =
		append(ih.matchData.matchIcons[ih.basicHandler.roundNumber-1][ih.basicHandler.frameGroup], newIcons...)

	var newHeaderTabular []string
	var newTabular []float64
	var tempHeader []string
	var tempData []float64
	var err error

	for _, tabularGenerator := range *ih.allTabularGenerators {
		tabularGenerator.Update()
		tempHeader, tempData, err = tabularGenerator.GetPeriodicTabularData()
		utils.CheckError(err)
		newHeaderTabular = append(newHeaderTabular, tempHeader...)
		newTabular = append(newTabular, tempData...)
	}

	if ih.isNewRound {
		ih.matchData.matchPeriodicTabularDataHeaders = nil
		ih.matchData.matchPeriodicTabularDataHeaders =
			append(ih.matchData.matchPeriodicTabularDataHeaders, newHeaderTabular...)
		ih.isNewRound = false
	}

	ih.matchData.matchPeriodicTabularData[ih.basicHandler.roundNumber-1][ih.basicHandler.frameGroup] =
		append(ih.matchData.matchPeriodicTabularData[ih.basicHandler.roundNumber-1][ih.basicHandler.frameGroup], newTabular...)

	ih.basicHandler.frameGroup = ih.basicHandler.frameGroup + 1
	parser := *(ih.basicHandler.parser)
	ih.lastUpdate = utils.GetRoundTime(parser, ih.basicHandler.roundStartTime, ih.basicHandler.tickRate)
}
