package composite_handlers

import (
	"github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
	utils "github.com/mrdbarros/csgo_analyze/utils"
)

type FlashUsageCalculator struct {
	statisticHolder
	blindPlayers map[uint64]flashInfo
}

type flashInfo struct {
	victim           *common.Player
	attacker         *common.Player
	projectile       *common.GrenadeProjectile
	duration         float64
	timeOfExplosion  float64
	blindnessEndtime float64
}

func (fc *FlashUsageCalculator) Register(bh *BasicHandler) error {
	fc.basicHandler = bh
	bh.RegisterRoundStartSubscriber(interface{}(fc).(RoundStartSubscriber))
	bh.RegisterFlashExplodeSubscriber(interface{}(fc).(FlashExplodeSubscriber))
	bh.RegisterKillSubscriber(interface{}(fc).(KillSubscriber))
	bh.RegisterPlayerFlashedSubscriber(interface{}(fc).(PlayerFlashedSubscriber))
	bh.RegisterRoundEndOfficialSubscriber(interface{}(fc).(RoundEndOfficialSubscriber))
	bh.RegisterRoundFreezetimeEndSubscriber(interface{}(fc).(RoundFreezetimeEndSubscriber))
	bh.RegisterRoundEndSubscriber(interface{}(fc).(RoundEndSubscriber))
	fc.baseStatsHeaders = []string{"Flashes Thrown", "Flashes Thrown_T", "Flashes Thrown_CT",
		"Enemies Blinded", "Enemies Blinded_T", "Enemies Blinded_CT",
		"Teammates Blinded", "Teammates Blinded_T", "Teammates Blinded_CT",
		"Total Enemy Blind Time", "Total Enemy Blind Time_T", "Total Enemy Blind Time_CT",
		"Total Teammate Blind Time", "Total Teammate Blind Time_T", "Total Teammate Blind Time_CT",
		"Flashes Leading To Enemy Death", "Flashes Leading To Enemy Death_T", "Flashes Leading To Enemy Death_CT",
		"Flashes Leading To Teammate Death", "Flashes Leading To Teammate Death_T", "Flashes Leading To Teammate Death_CT",
		"Net Players Blinded (Enemies-Teammates)", "Net Players Blinded (Enemies-Teammates)_T", "Net Players Blinded (Enemies-Teammates)_CT",
		"Net Flashes Leading To Death (Enemies-Teammates)", "Net Flashes Leading To Death (Enemies-Teammates)_T", "Net Flashes Leading To Death (Enemies-Teammates)_CT",
		"Net Blind Time (Enemies-Teammates)", "Net Blind Time (Enemies-Teammates)_T", "Net Blind Time (Enemies-Teammates)_CT",
	}
	fc.ratioStats = append(fc.ratioStats, [3]string{"Enemies Blinded Per Flash", "Enemies Blinded", "Flashes Thrown"})
	fc.ratioStats = append(fc.ratioStats, [3]string{"Teammates Blinded Per Flash", "Teammates Blinded", "Flashes Thrown"})
	fc.ratioStats = append(fc.ratioStats, [3]string{"Net Players Blinded Per Flash", "Net Players Blinded (Enemies-Teammates)", "Flashes Thrown"})
	fc.ratioStats = append(fc.ratioStats, [3]string{"Net Flashes Leading To Death Per Flash", "Net Flashes Leading To Death (Enemies-Teammates)", "Flashes Thrown"})
	fc.ratioStats = append(fc.ratioStats, [3]string{"Average Blind Time Per Flash", "Net Blind Time (Enemies-Teammates)", "Flashes Thrown"})
	fc.ratioStats = append(fc.ratioStats, [3]string{"Flashes Thrown Per Round", "Flashes Thrown", "Rounds"})
	fc.defaultValues = make(map[string]float64)
	fc.blindPlayers = make(map[uint64]flashInfo)
	return nil
}

func (fc *FlashUsageCalculator) RoundStartHandler(e events.RoundStart) {
	if fc.basicHandler.roundNumber-1 < len(fc.playerStats) {
		fc.playerStats = fc.playerStats[:fc.basicHandler.roundNumber-1]
	}
}

func (fc *FlashUsageCalculator) RoundFreezetimeEndHandler(e events.RoundFreezetimeEnd) {

	fc.AddNewRound()
}

func (fc *FlashUsageCalculator) FlashExplodeHandler(e events.FlashExplode) {

	if e.Thrower != nil {
		fc.addToPlayerStat(e.Thrower, 1, "Flashes Thrown")
	}

}

func (fc *FlashUsageCalculator) RoundEndHandler(e events.RoundEnd) {

	if fc.basicHandler.isMatchEnded {
		fc.processRoundEnd()
	}

}

func (fc *FlashUsageCalculator) processRoundEnd() {
	var statIndex int
	var playerID uint64
	var roundID int
	var enemiesIndex int
	var teammatesIndex int
	roundID = len(fc.playerStats) - 1
	for _, playerMapping := range fc.basicHandler.playerMappings[roundID] {
		playerID = playerMapping.playerObject.SteamID64
		for _, suffix := range [3]string{"", "_T", "_CT"} {

			statIndex = utils.IndexOf("Net Players Blinded (Enemies-Teammates)"+suffix, fc.baseStatsHeaders)
			enemiesIndex = utils.IndexOf("Enemies Blinded"+suffix, fc.baseStatsHeaders)
			teammatesIndex = utils.IndexOf("Teammates Blinded"+suffix, fc.baseStatsHeaders)
			fc.playerStats[roundID][playerID][statIndex] = fc.playerStats[roundID][playerID][enemiesIndex] - fc.playerStats[roundID][playerID][teammatesIndex]

			statIndex = utils.IndexOf("Net Flashes Leading To Death (Enemies-Teammates)"+suffix, fc.baseStatsHeaders)
			enemiesIndex = utils.IndexOf("Flashes Leading To Enemy Death"+suffix, fc.baseStatsHeaders)
			teammatesIndex = utils.IndexOf("Flashes Leading To Teammate Death"+suffix, fc.baseStatsHeaders)
			fc.playerStats[roundID][playerID][statIndex] = fc.playerStats[roundID][playerID][enemiesIndex] - fc.playerStats[roundID][playerID][teammatesIndex]

			statIndex = utils.IndexOf("Net Blind Time (Enemies-Teammates)"+suffix, fc.baseStatsHeaders)
			enemiesIndex = utils.IndexOf("Total Enemy Blind Time"+suffix, fc.baseStatsHeaders)
			teammatesIndex = utils.IndexOf("Total Teammate Blind Time"+suffix, fc.baseStatsHeaders)
			fc.playerStats[roundID][playerID][statIndex] = fc.playerStats[roundID][playerID][enemiesIndex] - fc.playerStats[roundID][playerID][teammatesIndex]

		}

	}
}

func (fc *FlashUsageCalculator) RoundEndOfficialHandler(e events.RoundEndOfficial) {
	if !fc.basicHandler.isMatchEnded {
		fc.processRoundEnd()
	}

}

func (fc *FlashUsageCalculator) KillHandler(e events.Kill) {

	if e.Victim != nil {

		if flashInfo, ok := fc.blindPlayers[e.Victim.SteamID64]; ok {
			if flashInfo.blindnessEndtime > fc.basicHandler.currentTime {
				if e.Victim.Team == flashInfo.attacker.Team {
					fc.addToPlayerStat(flashInfo.attacker, 1, "Flashes Leading To Teammate Death")
				} else {
					fc.addToPlayerStat(flashInfo.attacker, 1, "Flashes Leading To Enemy Death")
				}

			}
		}
	}

}

func (fc *FlashUsageCalculator) PlayerFlashedHandler(e events.PlayerFlashed) {
	var relevantFlashInfo bool
	duration := e.FlashDuration().Seconds()
	if finfo, ok := fc.blindPlayers[e.Player.SteamID64]; ok {
		if finfo.blindnessEndtime < duration+fc.basicHandler.currentTime {
			fc.blindPlayers[e.Player.SteamID64] = flashInfo{victim: e.Player, attacker: e.Attacker, projectile: e.Projectile,
				duration: duration, timeOfExplosion: fc.basicHandler.currentTime,
				blindnessEndtime: fc.basicHandler.currentTime + duration,
			}
			relevantFlashInfo = true

		}
	} else {

		fc.blindPlayers[e.Player.SteamID64] = flashInfo{victim: e.Player, attacker: e.Attacker, projectile: e.Projectile,
			duration: duration, timeOfExplosion: fc.basicHandler.currentTime,
			blindnessEndtime: fc.basicHandler.currentTime + duration,
		}

		relevantFlashInfo = true
	}

	if relevantFlashInfo {
		if e.Attacker.Team == e.Player.Team {
			fc.addToPlayerStat(e.Attacker, e.FlashDuration().Seconds(), "Total Teammate Blind Time")
			fc.addToPlayerStat(e.Attacker, 1, "Teammates Blinded")
		} else {
			fc.addToPlayerStat(e.Attacker, e.FlashDuration().Seconds(), "Total Enemy Blind Time")
			fc.addToPlayerStat(e.Attacker, 1, "Enemies Blinded")
		}
	}

}
