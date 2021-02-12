package composite_handlers

import (
	"github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	utils "github.com/mrdbarros/csgo_analyze/utils"
)

type statisticHolder struct {
	basicHandler        *BasicHandler
	baseStatsHeaders    []string
	playerStats         []map[uint64][]float64
	defaultValues       map[string]float64
	ratioStats          [][3]string
	consolidatedHeaders []string
	consolidatedStats   map[uint64][]float64
}

func (kc statisticHolder) GetRoundStatistic(roundNumber int, userID uint64) ([]string, []float64, error) {
	if roundNumber > len(kc.playerStats) {
		// fmt.Println("Invalid round number")
		return nil, nil, nil
	} else {
		return kc.baseStatsHeaders, kc.playerStats[roundNumber-1][userID], nil
	}

}

func (kc *statisticHolder) addToPlayerStat(player *common.Player, addAmount float64, stat string) {
	if _, ok := kc.playerStats[len(kc.playerStats)-1][player.SteamID64]; ok {
		isCT := (player.Team == 3)
		var suffix string
		kc.playerStats[len(kc.playerStats)-1][player.SteamID64][utils.IndexOf(stat, kc.baseStatsHeaders)] += addAmount
		if isCT {
			suffix = "_CT"
		} else {
			suffix = "_T"
		}
		if utils.IndexOf(stat+suffix, kc.baseStatsHeaders) != -1 {
			kc.playerStats[len(kc.playerStats)-1][player.SteamID64][utils.IndexOf(stat+suffix, kc.baseStatsHeaders)] += addAmount
		}
	}

}

func (kc *statisticHolder) setPlayerStat(player *common.Player, value float64, stat string) {
	if _, ok := kc.playerStats[len(kc.playerStats)-1][player.SteamID64]; ok {
		isCT := (player.Team == common.TeamCounterTerrorists)
		var suffix string
		roundID := len(kc.playerStats) - 1
		kc.playerStats[roundID][player.SteamID64][utils.IndexOf(stat, kc.baseStatsHeaders)] = value
		if isCT {
			suffix = "_CT"
		} else {
			suffix = "_T"
		}
		if utils.IndexOf(stat+suffix, kc.baseStatsHeaders) != -1 {
			kc.playerStats[roundID][player.SteamID64][utils.IndexOf(stat+suffix, kc.baseStatsHeaders)] = value
		}
	}
}

func (sh *statisticHolder) getPlayerStat(player *common.Player, stat string) float64 {
	return sh.playerStats[len(sh.playerStats)-1][player.SteamID64][utils.IndexOf(stat, sh.baseStatsHeaders)]
}

func (sh *statisticHolder) GetMatchStatistic(userID uint64) ([]string, []float64, error) {
	consolidatedStat := []float64{}
	// var ratioHeaders []string
	// var ratioStats []float64
	// var denominatorStat float64

	for _, roundStatMap := range sh.playerStats { //roundStatMap is map[uint64][]float64 of all base stats of the round
		if playerStat, ok := roundStatMap[userID]; ok { //get stats for specific player and round
			consolidatedStat = utils.ElementWiseSum(consolidatedStat, playerStat)
			// if userID == 76561197976402790 && sh.baseStatsHeaders[0] == "Rounds" {
			// 	fmt.Println("round", i+1, "rounds:", consolidatedStat[0], "rounds_t:", consolidatedStat[1], "rounds_ct", consolidatedStat[2])
			// }
		}

	}

	sh.consolidatedStats = make(map[uint64][]float64)
	// sh.consolidatedHeaders = append(sh.baseStatsHeaders, ratioHeaders...)
	// sh.consolidatedStats[userID] = append(consolidatedStat, ratioStats...)

	sh.consolidatedHeaders = sh.baseStatsHeaders
	sh.consolidatedStats[userID] = consolidatedStat

	return sh.consolidatedHeaders, sh.consolidatedStats[userID], nil
}

func (kc *statisticHolder) AddNewRound() {
	var newStats []float64
	kc.playerStats = append(kc.playerStats, make(map[uint64][]float64))

	for _, header := range kc.baseStatsHeaders {
		if val, ok := kc.defaultValues[header]; ok {
			newStats = append(newStats, val)
		} else {
			newStats = append(newStats, 0)
		}
	}
	for _, playerMapping := range kc.basicHandler.playerMappings[kc.basicHandler.roundNumber-1] {
		kc.playerStats[len(kc.playerStats)-1][playerMapping.playerObject.SteamID64] = make([]float64, len(kc.baseStatsHeaders))
		copy(kc.playerStats[len(kc.playerStats)-1][playerMapping.playerObject.SteamID64], newStats)
	}

}

func (kc *statisticHolder) GetRatioStatistics() [][3]string {
	return kc.basicHandler.ratioStats
}

type PlayerStatisticCalculator interface {
	CompositeEventHandler
	GetRoundStatistic(roundNumber int, userID uint64) ([]string, []float64, error) //stats header, stats
	GetMatchStatistic(userID uint64) ([]string, []float64, error)                  //stats header, stats
}
