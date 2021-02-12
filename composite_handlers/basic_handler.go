package composite_handlers

import (
	"fmt"
	"strconv"
	"time"

	dp "github.com/markus-wa/godispatch"

	dem "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
	metadata "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/metadata"
	utils "github.com/mrdbarros/csgo_analyze/utils"
)

//struct to gather player related data
type playerMapping struct {
	currentSlot  int
	playerObject *common.Player
}

//basic shared handler for parsing
type BasicHandler struct {
	parser      *dem.Parser
	tickRate    int
	mapMetadata metadata.Map
	statisticHolder

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

	roundEndOfficialHandlerID   dp.HandlerIdentifier
	roundEndOfficialSubscribers []RoundEndOfficialSubscriber

	bombDroppedHandlerID   dp.HandlerIdentifier
	bombDroppedSubscribers []BombDroppedSubscriber

	bombDefusedHandlerID   dp.HandlerIdentifier
	bombDefusedSubscribers []BombDefusedSubscriber

	flashExplodeHandlerID   dp.HandlerIdentifier
	flashExplodeSubscribers []FlashExplodeSubscriber

	bombPickupHandlerID   dp.HandlerIdentifier
	bombPickupSubscribers []BombPickupSubscriber

	footstepHandlerID   dp.HandlerIdentifier
	footstepSubscribers []FootstepSubscriber

	scoreUpdatedHandlerID   dp.HandlerIdentifier
	scoreUpdatedSubscribers []ScoreUpdatedSubscriber

	heExplodeHandlerID   dp.HandlerIdentifier
	heExplodeSubscribers []HeExplodeSubscriber

	itemDropHandlerID   dp.HandlerIdentifier
	itemDropSubscribers []ItemDropSubscriber

	itemPickupHandlerID   dp.HandlerIdentifier
	itemPickupSubscribers []ItemPickupSubscriber

	killHandlerID   dp.HandlerIdentifier
	killSubscribers []KillSubscriber

	playerFlashedHandlerID   dp.HandlerIdentifier
	playerFlashedSubscribers []PlayerFlashedSubscriber

	playerHurtHandlerID   dp.HandlerIdentifier
	playerHurtSubscribers []PlayerHurtSubscriber

	weaponReloadHandlerID   dp.HandlerIdentifier
	weaponReloadSubscribers []WeaponReloadSubscriber

	isWarmupPeriodChangedHandlerID   dp.HandlerIdentifier
	isWarmupPeriodChangedSubscribers []IsWarmupPeriodChangedSubscriber

	playerTeamChangeHandlerID   dp.HandlerIdentifier
	playerTeamChangeSubscribers []PlayerTeamChangeSubscriber

	playerDisconnectedHandlerID   dp.HandlerIdentifier
	playerDisconnectedSubscribers []PlayerDisconnectedSubscriber

	roundStartTime          float64
	currentTime             float64
	currentScore            string
	roundNumber             int
	frameGroup              int
	isMatchStarted          bool
	roundFreezeTime         bool
	roundWinner             string
	matchPointTeam          string
	isMatchEnded            bool
	isValidRoundStart       bool
	roundStructureCreated   bool
	terroristFirstTeamscore int
	ctFirstTeamScore        int
	fileName                string
	roundWinnerDetermined   bool
	roundProcessed          bool
	scoreUpdated            bool
	playerMappings          []map[uint64]playerMapping
	matchDatetime           time.Time
}

func (bh *BasicHandler) Register(basicHand *BasicHandler) error {

	return nil
}

func (bh *BasicHandler) Update() {

}

func (bh *BasicHandler) RegisterBasicEvents() error {
	parser := *(bh.parser)
	bh.roundStartHandlerID = parser.RegisterEventHandler(bh.RoundStartHandler)
	bh.roundEndHandlerID = parser.RegisterEventHandler(bh.RoundEndHandler)
	bh.roundFreezeTimeEndHandlerID = parser.RegisterEventHandler(bh.RoundFreezetimeEndHandler)
	bh.playerDisconnectedHandlerID = parser.RegisterEventHandler(bh.PlayerDisconnectedHandler)
	bh.scoreUpdatedHandlerID = parser.RegisterEventHandler(bh.ScoreUpdatedHandler)
	bh.footstepHandlerID = parser.RegisterEventHandler(bh.FootstepHandler)
	return nil
}

func (bh *BasicHandler) Setup(parser *dem.Parser, tickRate int, mapMetadata metadata.Map, matchDateTime time.Time, fileName string) error {
	bh.parser = parser
	bh.tickRate = tickRate
	bh.mapMetadata = mapMetadata
	bh.statisticHolder.baseStatsHeaders = []string{"Rounds", "Rounds_T", "Rounds_CT"}
	bh.basicHandler = bh
	bh.matchDatetime = matchDateTime
	bh.fileName = fileName

	return nil
}

func (bh *BasicHandler) UpdateTime() {
	bh.currentTime = utils.GetCurrentTime(*(bh.parser), bh.tickRate)
}

func (bh *BasicHandler) GetPeriodicTabularData() ([]string, []float64, error) {
	bh.UpdateTime()
	newCSVRow := []float64{0}
	currentRoundTime := bh.currentTime

	newCSVRow[0] = currentRoundTime - bh.roundStartTime
	header := []string{"round_time"}
	return header, newCSVRow, nil

}

func (bh *BasicHandler) RegisterRoundStartSubscriber(rs RoundStartSubscriber) {
	parser := *(bh.parser)
	if bh.roundStartHandlerID == nil {
		bh.roundStartHandlerID = parser.RegisterEventHandler(bh.RoundStartHandler)
	}

	bh.roundStartSubscribers = append(bh.roundStartSubscribers, rs)

}

//this a workaround for demos that update score after round start event (wtf)
func (bh *BasicHandler) createPreRoundStartInfo() {

	parser := *(bh.parser)
	bh.scoreUpdated = false
	gs := parser.GameState()
	bh.roundStructureCreated = false
	currentMappings := currentPlayerMappings(parser.GameState())

	bh.roundNumber = gs.TeamCounterTerrorists().Score() + gs.TeamTerrorists().Score() + 1
	if len(currentMappings) > 0 && !bh.isMatchEnded && !gs.IsWarmupPeriod() && bh.roundNumber-1 <= len(bh.playerMappings) {
		bh.isValidRoundStart = true
	} else {
		bh.isValidRoundStart = false
	}

	if bh.isValidRoundStart {
		bh.roundWinnerDetermined = false
		bh.roundFreezeTime = true
		bh.roundWinner = ""
		bh.frameGroup = 0
		bh.isMatchStarted = true
		tTeam := gs.TeamTerrorists()
		ctTeam := gs.TeamCounterTerrorists()

		scoreDiff := utils.Abs((tTeam.Score() - ctTeam.Score()))
		isTMatchPoint := (tTeam.Score() >= 15 && tTeam.Score()%3 == 0 && scoreDiff >= 1 && tTeam.Score() > ctTeam.Score())
		isCTMatchPoint := (ctTeam.Score() >= 15 && ctTeam.Score()%3 == 0 && scoreDiff >= 1 && ctTeam.Score() > tTeam.Score())
		if isTMatchPoint {
			bh.matchPointTeam = "t"
		} else if isCTMatchPoint {
			bh.matchPointTeam = "ct"
		} else {
			bh.matchPointTeam = ""
		}
		bh.currentScore = utils.PadLeft(strconv.Itoa(bh.roundNumber), "0", 2) + "_ct_" +
			utils.PadLeft(strconv.Itoa(gs.TeamCounterTerrorists().Score()), "0", 2) +
			"_t_" + utils.PadLeft(strconv.Itoa(gs.TeamTerrorists().Score()), "0", 2)
		if !bh.isMatchEnded && bh.isMatchStarted {
			if bh.roundNumber-1 < len(bh.playerMappings) {
				bh.CropData(bh.roundNumber - 1)
			}

			bh.playerMappings = append(bh.playerMappings, currentMappings)

			for _, subscriber := range bh.roundStartSubscribers {
				subscriber.RoundStartHandler(events.RoundStart{})
			}
		}

	}

}

func (bh *BasicHandler) RoundStartHandler(e events.RoundStart) {
	bh.UpdateTime()
	bh.roundStartTime = bh.currentTime
	if !bh.isMatchEnded && !bh.roundProcessed {
		bh.RoundEndOfficialHandler(events.RoundEndOfficial{})
	}
	bh.createPreRoundStartInfo()

}

func (bh *BasicHandler) getPlayersAlive(team common.Team) (playersAlive []*common.Player) {
	for _, playerMapping := range bh.playerMappings[bh.roundNumber-1] {
		player := playerMapping.playerObject
		if player.Team == team && player.IsAlive() {
			playersAlive = append(playersAlive, player)
		}
	}
	return playersAlive
}

func (bh *BasicHandler) CropData(index int) {
	bh.playerMappings = bh.playerMappings[:index]
}

func (bh *BasicHandler) RegisterRoundEndSubscriber(rs RoundEndSubscriber) {
	parser := *(bh.parser)
	if bh.roundEndHandlerID == nil {
		bh.roundEndHandlerID = parser.RegisterEventHandler(bh.RoundEndHandler)
	}

	bh.roundEndSubscribers = append(bh.roundEndSubscribers, rs)

}

func (bh *BasicHandler) RoundEndHandler(e events.RoundEnd) {
	bh.UpdateTime()
	if bh.roundStructureCreated && bh.isMatchStarted {
		for _, subscriber := range bh.roundEndSubscribers {
			subscriber.RoundEndHandler(e)
		}
	}

}

func (bh *BasicHandler) RegisterGrenadeEventIfSubscriber(rs GrenadeEventIfSubscriber) {
	parser := *(bh.parser)
	if bh.grenadeEventIfHandlerID == nil {
		bh.grenadeEventIfHandlerID = parser.RegisterEventHandler(bh.GrenadeEventIfHandler)
	}

	bh.grenadeEventIfSubscribers = append(bh.grenadeEventIfSubscribers, rs)

}

func (bh *BasicHandler) GrenadeEventIfHandler(e events.GrenadeEventIf) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted && bh.roundStructureCreated {
		for _, subscriber := range bh.grenadeEventIfSubscribers {
			subscriber.GrenadeEventIfHandler(e)
		}
	}
}

func (bh *BasicHandler) RegisterRoundFreezetimeEndSubscriber(rs RoundFreezetimeEndSubscriber) {
	parser := *(bh.parser)
	if bh.roundFreezeTimeEndHandlerID == nil {
		bh.roundFreezeTimeEndHandlerID = parser.RegisterEventHandler(bh.RoundFreezetimeEndHandler)
	}

	bh.roundFreezeTimeEndSubscribers = append(bh.roundFreezeTimeEndSubscribers, rs)

}

//this is a workaround for replays that don't send freezetimeend event sometimes
func (bh *BasicHandler) createRoundStructure() {

	if bh.roundFreezeTime && bh.isValidRoundStart {
		bh.roundFreezeTime = false
		bh.roundProcessed = false
		if len(bh.playerMappings[bh.roundNumber-1]) > 0 && !bh.isMatchEnded {
			bh.roundStructureCreated = true
		} else {
			bh.roundStructureCreated = false
		}
		parser := (*bh.parser)
		currentMappings := currentPlayerMappings(parser.GameState())
		bh.playerMappings[len(bh.playerMappings)-1] = currentMappings
		if bh.roundNumber-1 < len(bh.playerStats) {
			bh.playerStats = bh.playerStats[:bh.roundNumber-1]
		}
		bh.AddNewRound() //adds new round for statistic holder

		for _, player := range bh.playerMappings[bh.roundNumber-1] {
			bh.statisticHolder.setPlayerStat(player.playerObject, 1, "Rounds")
		}

		for _, subscriber := range bh.roundFreezeTimeEndSubscribers {
			subscriber.RoundFreezetimeEndHandler(events.RoundFreezetimeEnd{})
		}
	}

}

func (bh *BasicHandler) RoundFreezetimeEndHandler(e events.RoundFreezetimeEnd) {
	bh.UpdateTime()

	// if !bh.isMatchEnded && bh.isMatchStarted && bh.isValidRoundStart {

	// 	bh.createRoundStructure()
	// }
}

func (bh *BasicHandler) RegisterBombPlantedSubscriber(rs BombPlantedSubscriber) {
	parser := *(bh.parser)
	if bh.bombPlantedHandlerID == nil {
		bh.bombPlantedHandlerID = parser.RegisterEventHandler(bh.BombPlantedHandler)
	}

	bh.bombPlantedSubscribers = append(bh.bombPlantedSubscribers, rs)

}

func (bh *BasicHandler) BombPlantedHandler(e events.BombPlanted) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted && bh.roundStructureCreated {
		for _, subscriber := range bh.bombPlantedSubscribers {
			subscriber.BombPlantedHandler(e)
		}
	}
}

func (bh *BasicHandler) RegisterFrameDoneSubscriber(rs FrameDoneSubscriber) {
	parser := *(bh.parser)
	if bh.frameDoneHandlerID == nil {
		bh.frameDoneHandlerID = parser.RegisterEventHandler(bh.FrameDoneHandler)
	}

	bh.frameDoneSubscribers = append(bh.frameDoneSubscribers, rs)

}

func (bh *BasicHandler) FrameDoneHandler(e events.FrameDone) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted && bh.roundStructureCreated {
		for _, subscriber := range bh.frameDoneSubscribers {
			subscriber.FrameDoneHandler(e)
		}
	}
}

func (bh *BasicHandler) RegisterRoundEndOfficialSubscriber(rs RoundEndOfficialSubscriber) {
	parser := *(bh.parser)
	if bh.roundEndOfficialHandlerID == nil {
		bh.roundEndOfficialHandlerID = parser.RegisterEventHandler(bh.RoundEndOfficialHandler)
	}

	bh.roundEndOfficialSubscribers = append(bh.roundEndOfficialSubscribers, rs)

}

func (bh *BasicHandler) RoundEndOfficialHandler(e events.RoundEndOfficial) {
	bh.UpdateTime()
	if bh.isMatchStarted && bh.roundStructureCreated {
		for _, subscriber := range bh.roundEndOfficialSubscribers {
			subscriber.RoundEndOfficialHandler(e)
		}
		bh.roundProcessed = true
	}
}

func (bh *BasicHandler) RegisterBombDroppedSubscriber(rs BombDroppedSubscriber) {
	parser := *(bh.parser)
	if bh.bombDroppedHandlerID == nil {
		bh.bombDroppedHandlerID = parser.RegisterEventHandler(bh.BombDroppedHandler)
	}

	bh.bombDroppedSubscribers = append(bh.bombDroppedSubscribers, rs)

}

func (bh *BasicHandler) BombDroppedHandler(e events.BombDropped) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted && bh.roundStructureCreated {
		for _, subscriber := range bh.bombDroppedSubscribers {
			subscriber.BombDroppedHandler(e)
		}
	}
}

func (bh *BasicHandler) RegisterBombDefusedSubscriber(rs BombDefusedSubscriber) {
	parser := *(bh.parser)
	if bh.bombDefusedHandlerID == nil {
		bh.bombDefusedHandlerID = parser.RegisterEventHandler(bh.BombDefusedHandler)
	}

	bh.bombDefusedSubscribers = append(bh.bombDefusedSubscribers, rs)

}

func (bh *BasicHandler) BombDefusedHandler(e events.BombDefused) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted && bh.roundStructureCreated {
		for _, subscriber := range bh.bombDefusedSubscribers {
			subscriber.BombDefusedHandler(e)
		}
	}
}

func (bh *BasicHandler) RegisterFlashExplodeSubscriber(rs FlashExplodeSubscriber) {
	parser := *(bh.parser)
	if bh.flashExplodeHandlerID == nil {
		bh.flashExplodeHandlerID = parser.RegisterEventHandler(bh.FlashExplodeHandler)
	}

	bh.flashExplodeSubscribers = append(bh.flashExplodeSubscribers, rs)

}

func (bh *BasicHandler) FlashExplodeHandler(e events.FlashExplode) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted && bh.roundStructureCreated {
		for _, subscriber := range bh.flashExplodeSubscribers {
			subscriber.FlashExplodeHandler(e)
		}
	}
}

func (bh *BasicHandler) RegisterBombPickupSubscriber(rs BombPickupSubscriber) {
	parser := *(bh.parser)
	if bh.bombPickupHandlerID == nil {
		bh.bombPickupHandlerID = parser.RegisterEventHandler(bh.BombPickupHandler)
	}

	bh.bombPickupSubscribers = append(bh.bombPickupSubscribers, rs)

}

func (bh *BasicHandler) BombPickupHandler(e events.BombPickup) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted && bh.roundStructureCreated {
		for _, subscriber := range bh.bombPickupSubscribers {
			subscriber.BombPickupHandler(e)
		}
	}
}

func (bh *BasicHandler) RegisterFootstepSubscriber(rs FootstepSubscriber) {
	parser := *(bh.parser)
	if bh.footstepHandlerID == nil {
		bh.footstepHandlerID = parser.RegisterEventHandler(bh.FootstepHandler)
	}

	bh.footstepSubscribers = append(bh.footstepSubscribers, rs)

}

func (bh *BasicHandler) FootstepHandler(e events.Footstep) {
	bh.UpdateTime()
	if bh.isValidRoundStart && !bh.roundWinnerDetermined && !bh.roundStructureCreated {

		bh.createRoundStructure()
	} else if bh.isValidRoundStart && bh.scoreUpdated && !bh.roundWinnerDetermined {
		bh.createPreRoundStartInfo()
		if !bh.roundStructureCreated {
			bh.createRoundStructure()
		}

	}
	if !bh.isMatchEnded && bh.isMatchStarted && bh.roundStructureCreated {
		for _, subscriber := range bh.footstepSubscribers {
			subscriber.FootstepHandler(e)
		}
	}
}

func (bh *BasicHandler) RegisterScoreUpdatedSubscriber(rs ScoreUpdatedSubscriber) {
	parser := *(bh.parser)
	if bh.scoreUpdatedHandlerID == nil {
		bh.scoreUpdatedHandlerID = parser.RegisterEventHandler(bh.ScoreUpdatedHandler)
	}

	bh.scoreUpdatedSubscribers = append(bh.scoreUpdatedSubscribers, rs)

}

func (bh *BasicHandler) ScoreUpdatedHandler(e events.ScoreUpdated) {
	bh.UpdateTime()
	bh.scoreUpdated = true
	if !bh.roundWinnerDetermined && bh.roundStructureCreated {
		winTeam := e.TeamState.Team()
		tPoint := 0
		ctPoint := 0
		bh.isValidRoundStart = false
		if winTeam == common.TeamTerrorists {
			bh.roundWinner = "t"
			tPoint += 1
		} else if winTeam == common.TeamCounterTerrorists {
			bh.roundWinner = "ct"
			ctPoint += 1
		} else {
			bh.roundWinner = "invalid"
		}

		gs := (*bh.parser).GameState()
		if bh.roundNumber > 15 {
			bh.terroristFirstTeamscore = gs.TeamCounterTerrorists().Score() + ctPoint
			bh.ctFirstTeamScore = gs.TeamTerrorists().Score() + tPoint
		} else {
			bh.terroristFirstTeamscore = gs.TeamTerrorists().Score() + tPoint
			bh.ctFirstTeamScore = gs.TeamCounterTerrorists().Score() + ctPoint
		}
		if bh.roundWinner == bh.matchPointTeam && bh.matchPointTeam != "" && bh.isMatchStarted {
			bh.isMatchEnded = true
		}
		if bh.isMatchStarted {
			for _, subscriber := range bh.scoreUpdatedSubscribers {
				subscriber.ScoreUpdatedHandler(e)
			}
		}
		bh.roundWinnerDetermined = true

		if bh.isMatchEnded {
			bh.RoundEndOfficialHandler(events.RoundEndOfficial{})
		}
	}

}

func (bh *BasicHandler) RegisterHeExplodeSubscriber(rs HeExplodeSubscriber) {
	parser := *(bh.parser)
	if bh.heExplodeHandlerID == nil {
		bh.heExplodeHandlerID = parser.RegisterEventHandler(bh.HeExplodeHandler)
	}

	bh.heExplodeSubscribers = append(bh.heExplodeSubscribers, rs)

}

func (bh *BasicHandler) HeExplodeHandler(e events.HeExplode) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted && bh.roundStructureCreated {
		for _, subscriber := range bh.heExplodeSubscribers {
			subscriber.HeExplodeHandler(e)
		}
	}
}

func (bh *BasicHandler) RegisterItemDropSubscriber(rs ItemDropSubscriber) {
	parser := *(bh.parser)
	if bh.itemDropHandlerID == nil {
		bh.itemDropHandlerID = parser.RegisterEventHandler(bh.ItemDropHandler)
	}

	bh.itemDropSubscribers = append(bh.itemDropSubscribers, rs)

}

func (bh *BasicHandler) ItemDropHandler(e events.ItemDrop) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted && bh.roundStructureCreated {
		for _, subscriber := range bh.itemDropSubscribers {
			subscriber.ItemDropHandler(e)
		}
	}
}

func (bh *BasicHandler) RegisterItemPickupSubscriber(rs ItemPickupSubscriber) {
	parser := *(bh.parser)
	if bh.itemPickupHandlerID == nil {
		bh.itemPickupHandlerID = parser.RegisterEventHandler(bh.ItemPickupHandler)
	}

	bh.itemPickupSubscribers = append(bh.itemPickupSubscribers, rs)

}

func (bh *BasicHandler) ItemPickupHandler(e events.ItemPickup) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted && bh.roundStructureCreated {
		for _, subscriber := range bh.itemPickupSubscribers {
			subscriber.ItemPickupHandler(e)
		}
	}
}

func (bh *BasicHandler) RegisterKillSubscriber(rs KillSubscriber) {
	parser := *(bh.parser)
	if bh.killHandlerID == nil {
		bh.killHandlerID = parser.RegisterEventHandler(bh.KillHandler)
	}

	bh.killSubscribers = append(bh.killSubscribers, rs)

}

func (bh *BasicHandler) KillHandler(e events.Kill) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted && bh.roundStructureCreated {
		for _, subscriber := range bh.killSubscribers {
			subscriber.KillHandler(e)
		}
	}
}

func (bh *BasicHandler) RegisterPlayerFlashedSubscriber(rs PlayerFlashedSubscriber) {
	parser := *(bh.parser)
	if bh.playerFlashedHandlerID == nil {
		bh.playerFlashedHandlerID = parser.RegisterEventHandler(bh.PlayerFlashedHandler)
	}

	bh.playerFlashedSubscribers = append(bh.playerFlashedSubscribers, rs)

}

func (bh *BasicHandler) PlayerFlashedHandler(e events.PlayerFlashed) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted && bh.roundStructureCreated {
		for _, subscriber := range bh.playerFlashedSubscribers {
			subscriber.PlayerFlashedHandler(e)
		}
	}
}

func (bh *BasicHandler) RegisterPlayerHurtSubscriber(rs PlayerHurtSubscriber) {
	parser := *(bh.parser)
	if bh.playerHurtHandlerID == nil {
		bh.playerHurtHandlerID = parser.RegisterEventHandler(bh.PlayerHurtHandler)
	}

	bh.playerHurtSubscribers = append(bh.playerHurtSubscribers, rs)
}

func (bh *BasicHandler) PlayerHurtHandler(e events.PlayerHurt) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted && bh.roundStructureCreated {
		for _, subscriber := range bh.playerHurtSubscribers {
			subscriber.PlayerHurtHandler(e)
		}
	}
}

func (bh *BasicHandler) RegisterWeaponReloadSubscriber(rs WeaponReloadSubscriber) {
	parser := *(bh.parser)
	if bh.weaponReloadHandlerID == nil {
		bh.weaponReloadHandlerID = parser.RegisterEventHandler(bh.WeaponReloadHandler)
	}

	bh.weaponReloadSubscribers = append(bh.weaponReloadSubscribers, rs)

}

func (bh *BasicHandler) WeaponReloadHandler(e events.WeaponReload) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted && bh.roundStructureCreated {
		for _, subscriber := range bh.weaponReloadSubscribers {
			subscriber.WeaponReloadHandler(e)
		}
	}
}

func (bh *BasicHandler) RegisterIsWarmupPeriodChangedSubscriber(rs IsWarmupPeriodChangedSubscriber) {
	parser := *(bh.parser)
	if bh.isWarmupPeriodChangedHandlerID == nil {
		bh.isWarmupPeriodChangedHandlerID = parser.RegisterEventHandler(bh.IsWarmupPeriodChangedHandler)
	}

	bh.isWarmupPeriodChangedSubscribers = append(bh.isWarmupPeriodChangedSubscribers, rs)

}

func (bh *BasicHandler) IsWarmupPeriodChangedHandler(e events.IsWarmupPeriodChanged) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted && bh.roundStructureCreated {
		for _, subscriber := range bh.isWarmupPeriodChangedSubscribers {
			subscriber.IsWarmupPeriodChangedHandler(e)
		}
	}
}

func (bh *BasicHandler) RegisterPlayerTeamChangeSubscriber(rs PlayerTeamChangeSubscriber) {
	parser := *(bh.parser)
	if bh.playerTeamChangeHandlerID == nil {
		bh.playerTeamChangeHandlerID = parser.RegisterEventHandler(bh.PlayerTeamChangeHandler)
	}

	bh.playerTeamChangeSubscribers = append(bh.playerTeamChangeSubscribers, rs)

}

func (bh *BasicHandler) PlayerTeamChangeHandler(e events.PlayerTeamChange) {
	bh.UpdateTime()

	if !bh.isMatchEnded && bh.isMatchStarted && bh.isValidRoundStart {

		// if e.NewTeam == common.TeamCounterTerrorists || e.NewTeam == common.TeamTerrorists {
		// 	if _, ok := bh.playerMappings[bh.roundNumber-1][e.Player.SteamID64]; !ok {
		// 		bh.playerMappings[bh.roundNumber-1] = currentPlayerMappings((*bh.parser).GameState())
		// 	}
		// }
		for _, subscriber := range bh.playerTeamChangeSubscribers {
			subscriber.PlayerTeamChangeHandler(e)
		}
	}
}

func (bh *BasicHandler) RegisterPlayerDisconnectedSubscriber(rs PlayerDisconnectedSubscriber) {
	parser := *(bh.parser)
	if bh.playerDisconnectedHandlerID == nil {
		bh.playerDisconnectedHandlerID = parser.RegisterEventHandler(bh.PlayerDisconnectedHandler)
	}

	bh.playerDisconnectedSubscribers = append(bh.playerDisconnectedSubscribers, rs)

}

func (bh *BasicHandler) PlayerDisconnectedHandler(e events.PlayerDisconnected) {
	bh.UpdateTime()

	if !bh.isMatchEnded && bh.isMatchStarted && bh.isValidRoundStart {

		for _, subscriber := range bh.playerDisconnectedSubscribers {
			subscriber.PlayerDisconnectedHandler(e)
		}
	}
}

func currentPlayerMappings(gs dem.GameState) map[uint64]playerMapping {
	newAllPlayers := make(map[uint64]playerMapping)
	players := gs.Participants().Playing()
	ctCount := 0
	tCount := 0
	playerBasePos := 0
	for _, player := range players {
		isCT := (player.Team == 3)
		isTR := (player.Team == 2)

		if isTR && tCount > 4 || isCT && ctCount > 4 {
			fmt.Println("invalid team size")
			return make(map[uint64]playerMapping)
		}

		if !(isCT || isTR) {
			fmt.Println("invalid team")
			break
		}

		if isCT {
			playerBasePos = ctCount + 5
			ctCount++
		} else if isTR {
			playerBasePos = tCount
			tCount++
		}
		newAllPlayers[player.SteamID64] = playerMapping{playerObject: player, currentSlot: playerBasePos}
	}

	return newAllPlayers
}
