package composite_handlers

import (
	"strconv"

	"github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
	utils "github.com/mrdbarros/csgo_analyze/utils"
)

type KDATCalculator struct {
	statisticHolder
	killsToBeTraded    map[uint64][]KillToBeTraded //maps from killerID to a list of their kills
	tradeIntervalLimit float64
	isFirstDuel        bool
	clutchSituations   []clutchSituation
}

type KillToBeTraded struct {
	killer      *common.Player
	victim      *common.Player
	timeOfDeath float64
}

func (kc *KDATCalculator) processKillTradeInformation(e events.Kill) {
	if e.Killer != nil {
		victimID := e.Victim.SteamID64
		var timeFromKill float64

		currentTime := kc.basicHandler.currentTime
		if victimKills, ok := kc.killsToBeTraded[victimID]; ok {
			for _, victimKill := range victimKills {
				timeFromKill = currentTime - victimKill.timeOfDeath
				if timeFromKill < kc.tradeIntervalLimit {
					kc.addToPlayerStat(victimKill.victim, 1, "Was Traded")
					kc.addToPlayerStat(e.Killer, 1, "Trades")
				}
			}

		}
		kc.addPlayerToBeTraded(e.Killer, e.Victim, currentTime)
	}

}

func (kc *KDATCalculator) addPlayerToBeTraded(killer *common.Player, victim *common.Player, timeOfDeath float64) {
	kc.killsToBeTraded[killer.SteamID64] = append(kc.killsToBeTraded[killer.SteamID64],
		KillToBeTraded{killer: killer, victim: victim, timeOfDeath: timeOfDeath})
}

func (kc *KDATCalculator) Setup(tradeIntervalLimit float64) {
	kc.tradeIntervalLimit = tradeIntervalLimit
}

func (kc *KDATCalculator) Register(bh *BasicHandler) error {
	kc.basicHandler = bh
	bh.RegisterRoundStartSubscriber(interface{}(kc).(RoundStartSubscriber))
	bh.RegisterKillSubscriber(interface{}(kc).(KillSubscriber))
	bh.RegisterRoundEndOfficialSubscriber(interface{}(kc).(RoundEndOfficialSubscriber))
	bh.RegisterRoundFreezetimeEndSubscriber(interface{}(kc).(RoundFreezetimeEndSubscriber))
	kc.baseStatsHeaders = []string{"Kills", "Kills_CT", "Kills_T",
		"Assists", "Assists_T", "Assists_CT",
		"Deaths", "Deaths_T", "Deaths_CT",
		"Trades", "Trades_T", "Trades_CT",
		"Was Traded", "Was Traded_T", "Was Traded_CT",
		"KAST Sum", "KAST Sum_T", "KAST Sum_CT",
		"1K", "1K_T", "1K_CT",
		"2K", "2K_T", "2K_CT",
		"3K", "3K_T", "3K_CT",
		"4K", "4K_T", "4K_CT",
		"5K", "5K_T", "5K_CT",
		"Multikills", "Multikills_T", "Multikills_CT",
		"First Kills", "First Kills_T", "First Kills_CT",
		"First Kill Attempts", "First Kill Attempts_T", "First Kill Attempts_CT",
		"Clutches", "Clutches_T", "Clutches_CT",
		"Clutch Attempts", "Clutch Attempts_T", "Clutch Attempts_CT",
		"1v1 Wins", "1v1 Wins_T", "1v1 Wins_CT",
		"1v1 Attempts", "1v1 Attempts_T", "1v1 Attempts_CT",
		"1v2 Wins", "1v2 Wins_T", "1v2 Wins_CT",
		"1v2 Attempts", "1v2 Attempts_T", "1v2 Attempts_CT",
		"1v3 Wins", "1v3 Wins_T", "1v3 Wins_CT",
		"1v3 Attempts", "1v3 Attempts_T", "1v3 Attempts_CT",
		"1v4 Wins", "1v4 Wins_T", "1v4 Wins_CT",
		"1v4 Attempts", "1v4 Attempts_T", "1v4 Attempts_CT",
		"1v5 Wins", "1v5 Wins_T", "1v5 Wins_CT",
		"1v5 Attempts", "1v5 Attempts_T", "1v5 Attempts_CT",
		"HS Kills", "HS Kills_T", "HS Kills_CT",
	}

	kc.defaultValues = make(map[string]float64)
	return nil
}

func (kc *KDATCalculator) RoundStartHandler(e events.RoundStart) {

}

func (kc *KDATCalculator) RoundFreezetimeEndHandler(e events.RoundFreezetimeEnd) {
	if kc.basicHandler.roundNumber-1 < len(kc.playerStats) {
		kc.playerStats = kc.playerStats[:kc.basicHandler.roundNumber-1]
	}

	kc.killsToBeTraded = make(map[uint64][]KillToBeTraded)
	kc.isFirstDuel = true
	kc.clutchSituations = nil
	kc.AddNewRound()
}

func (kc *KDATCalculator) processClutchSituation(winnerTeamName string) {
	var winnerTeam common.Team
	if winnerTeamName == "t" {
		winnerTeam = common.TeamTerrorists
	} else if winnerTeamName == "ct" {
		winnerTeam = common.TeamCounterTerrorists
	}
	//check for clutch
	var numberOfOpponents string
	for _, clutchSituation := range kc.clutchSituations {
		numberOfOpponents = strconv.Itoa(len(clutchSituation.opponents))
		kc.setPlayerStat(clutchSituation.clutcher, 1, "Clutch Attempts")
		kc.setPlayerStat(clutchSituation.clutcher, 1, "1v"+numberOfOpponents+" Attempts")
		if clutchSituation.clutcher.Team == winnerTeam {
			kc.setPlayerStat(clutchSituation.clutcher, 1, "Clutches")

			kc.setPlayerStat(clutchSituation.clutcher, 1, "1v"+numberOfOpponents+" Wins")
		}
	}

}

func (kc *KDATCalculator) RoundEndOfficialHandler(e events.RoundEndOfficial) {

	kc.processRoundEnd()
}

func (kc *KDATCalculator) processRoundEnd() {
	var playerKills float64
	var playerAssists float64
	var playerDeath float64
	var playerWasTraded float64
	var stringKills string
	kc.processClutchSituation(kc.basicHandler.roundWinner)
	roundID := len(kc.playerStats) - 1

	for _, playerMapping := range kc.basicHandler.playerMappings[roundID] {
		playerKills = kc.playerStats[roundID][playerMapping.playerObject.SteamID64][utils.IndexOf("Kills", kc.baseStatsHeaders)]
		playerAssists = kc.playerStats[roundID][playerMapping.playerObject.SteamID64][utils.IndexOf("Assists", kc.baseStatsHeaders)]
		playerDeath = kc.playerStats[roundID][playerMapping.playerObject.SteamID64][utils.IndexOf("Deaths", kc.baseStatsHeaders)]
		playerWasTraded = kc.playerStats[roundID][playerMapping.playerObject.SteamID64][utils.IndexOf("Was Traded", kc.baseStatsHeaders)]

		if playerKills > 0 || playerAssists > 0 || playerDeath == 0 || playerWasTraded > 0 {
			kc.setPlayerStat(playerMapping.playerObject, 1, "KAST Sum")
		}

		if playerKills > 0 {
			stringKills = strconv.FormatFloat(playerKills, 'f', -1, 64)
			kc.setPlayerStat(playerMapping.playerObject, 1, stringKills+"K")
		}

		if playerKills > 1 {
			kc.setPlayerStat(playerMapping.playerObject, 1, "Multikills")
		}

	}
}

// func (kc *KDATCalculator) RoundEndOfficialHandler(e events.RoundEndOfficial) {
// 	if !kc.basicHandler.isMatchEnded {
// 		kc.processRoundEnd()
// 	}

// }

func (kc *KDATCalculator) addKDAInfo(e events.Kill) {
	var addAmmount float64
	if e.Killer != nil {
		if e.Killer.Team != e.Victim.Team {
			addAmmount = 1
			if e.IsHeadshot {
				kc.addToPlayerStat(e.Killer, 1, "HS Kills")
			}
		} else {
			addAmmount = -1
		}
		kc.addToPlayerStat(e.Killer, addAmmount, "Kills")

	}

	if e.Assister != nil {
		if e.Assister.Team != e.Victim.Team {
			kc.addToPlayerStat(e.Assister, 1, "Assists")
		}

	}

	if e.Victim != nil {
		kc.addToPlayerStat(e.Victim, 1, "Deaths")
		kc.addDeath(e.Victim)
		if e.Killer == nil && e.Weapon.Type != common.EqBomb {
			kc.addToPlayerStat(e.Victim, -1, "Kills")
		}
	}

}

type clutchSituation struct {
	clutcher  *common.Player
	opponents []*common.Player
}

func (kc *KDATCalculator) addDeath(victim *common.Player) {
	remainingPlayers := kc.basicHandler.getPlayersAlive(victim.Team)
	remainingPlayers = RemovePlayerFromSlice(remainingPlayers, victim)

	remainingOpponents := kc.basicHandler.getPlayersAlive(victim.TeamState.Opponent.Team())
	if len(remainingPlayers) == 1 && len(remainingOpponents) > 0 {
		clutchSituation := clutchSituation{clutcher: remainingPlayers[0],
			opponents: kc.basicHandler.getPlayersAlive(victim.TeamState.Opponent.Team())}
		kc.clutchSituations = append(kc.clutchSituations, clutchSituation)
	}

}

func (kc *KDATCalculator) addFirstDuelInfo(e events.Kill) {
	if kc.isFirstDuel && e.Killer != nil && e.Victim != nil {
		if e.Killer.Team != e.Victim.Team {
			kc.setPlayerStat(e.Killer, 1, "First Kills")
			kc.setPlayerStat(e.Killer, 1, "First Kill Attempts")
			kc.setPlayerStat(e.Victim, 1, "First Kill Attempts")
			kc.isFirstDuel = false
		}

	}

}

func (kc *KDATCalculator) KillHandler(e events.Kill) {

	kc.addKDAInfo(e)
	kc.addFirstDuelInfo(e)

	kc.processKillTradeInformation(e)

}

func RemovePlayerFromSlice(s []*common.Player, player *common.Player) (returnSlice []*common.Player) {
	var removed bool
	for i, innerPlayer := range s {
		if player == innerPlayer {
			s[i] = s[len(s)-1]
			removed = true
		}
	}
	if removed {
		returnSlice = s[:len(s)-1]
	} else {
		returnSlice = s
	}
	return returnSlice
}
