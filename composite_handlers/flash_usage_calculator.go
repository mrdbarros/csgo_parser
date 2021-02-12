package composite_handlers

import (
	"github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/sendtables"
	map_builder "github.com/mrdbarros/csgo_analyze/map_builder"
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
	bh.RegisterRoundFreezetimeEndSubscriber(interface{}(fc).(RoundFreezetimeEndSubscriber))
	bh.RegisterRoundEndOfficialSubscriber(interface{}(fc).(RoundEndOfficialSubscriber))
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

	fc.defaultValues = make(map[string]float64)
	fc.blindPlayers = make(map[uint64]flashInfo)
	return nil
}

func (fc *FlashUsageCalculator) RoundStartHandler(e events.RoundStart) {

}

func (fc *FlashUsageCalculator) RoundFreezetimeEndHandler(e events.RoundFreezetimeEnd) {
	if fc.basicHandler.roundNumber-1 < len(fc.playerStats) {
		fc.playerStats = fc.playerStats[:fc.basicHandler.roundNumber-1]
	}
	fc.AddNewRound()
}

func (fc *FlashUsageCalculator) FlashExplodeHandler(e events.FlashExplode) {

	if e.Thrower != nil {
		fc.addToPlayerStat(e.Thrower, 1, "Flashes Thrown")
	}

}

func (fc *FlashUsageCalculator) RoundEndOfficialHandler(e events.RoundEndOfficial) {

	fc.processRoundEnd()

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

// func (fc *FlashUsageCalculator) RoundEndOfficialHandler(e events.RoundEndOfficial) {
// 	if !fc.basicHandler.isMatchEnded {
// 		fc.processRoundEnd()
// 	}

// }

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

func (fc *FlashUsageCalculator) Update() {
}

func (fc *FlashUsageCalculator) GetPeriodicIcons() ([]map_builder.Icon, error) {
	var flashIcons []map_builder.Icon
	var flashTimeLeft float64
	var flashIcon map_builder.Icon
	var playerMaskIcon map_builder.Icon
	var opacity int
	var projectileMap map[sendtables.Entity]int
	var suffix string
	projectileMap = make(map[sendtables.Entity]int)
	for _, flashInfo := range fc.blindPlayers {
		if flashInfo.blindnessEndtime >= fc.basicHandler.currentTime && flashInfo.victim.IsAlive() {
			flashTimeLeft = flashInfo.blindnessEndtime - fc.basicHandler.currentTime

			if _, ok := projectileMap[flashInfo.attacker.Entity]; !ok {
				flashIcon = map_builder.Icon{X: flashInfo.projectile.Position().X,
					Y:        flashInfo.projectile.Position().Y,
					IconName: "flashbang"}
				flashIcons = append(flashIcons, flashIcon)
				if flashInfo.attacker.Team == common.TeamCounterTerrorists {
					suffix = "_ct"
				} else if flashInfo.attacker.Team == common.TeamTerrorists {
					suffix = "_t"
				}
				flashIcon = map_builder.Icon{X: flashInfo.projectile.Position().X,
					Y:        flashInfo.projectile.Position().Y,
					IconName: "flashbang" + suffix}
				flashIcons = append(flashIcons, flashIcon)
				projectileMap[flashInfo.projectile.Entity] = 0
			}
			opacity = int((flashTimeLeft / 6.) * 255)
			if opacity > 255 {
				opacity = 255
			}
			playerMaskIcon = map_builder.Icon{X: flashInfo.victim.LastAlivePosition.X,
				Y:        flashInfo.victim.LastAlivePosition.Y,
				IconName: "flashbang_mask",
				Opacity:  opacity}
			flashIcons = append(flashIcons, playerMaskIcon)
		}
	}
	return flashIcons, nil
}
