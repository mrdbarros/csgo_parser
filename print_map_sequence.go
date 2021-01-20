package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"flag"
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"

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
	currentSlot  int
	playerObject *common.Player
}

//generic event handler registering interface
type CompositeEventHandler interface {
	Register(*basicHandler) error
	//Unregister() error
}

type PeriodicGenerator interface {
	CompositeEventHandler
	Update()
}

//IconGenerators generate icons on output map
type PeriodicIconGenerator interface {
	PeriodicGenerator
	GetPeriodicIcons() ([]utils.Icon, error)
}

//TabularGenerators generate data rows on output file
type PeriodicTabularGenerator interface {
	PeriodicGenerator
	GetPeriodicTabularData() ([]string, []float64, error) //header, data, error
}

//StatGenerators generate rows on output stat file
type StatGenerator interface {
	GetStatistics() ([]string, []float64, error) //header, data, error
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

//Interface to RoundEndOfficial event subscribers
type RoundEndOfficialSubscriber interface {
	RoundEndOfficialHandler(events.RoundEndOfficial)
}

//Interface to BombDropped event subscribers
type BombDroppedSubscriber interface {
	BombDroppedHandler(events.BombDropped)
}

//Interface to BombDefused event subscribers
type BombDefusedSubscriber interface {
	BombDefusedHandler(events.BombDefused)
}

//Interface to BombPickup event subscribers
type BombPickupSubscriber interface {
	BombPickupHandler(events.BombPickup)
}

//Interface to FlashExplode event subscribers
type FlashExplodeSubscriber interface {
	FlashExplodeHandler(events.FlashExplode)
}

//Interface to Footstep event subscribers
type FootstepSubscriber interface {
	FootstepHandler(events.Footstep)
}

//Interface to ScoreUpdated event subscribers
type ScoreUpdatedSubscriber interface {
	ScoreUpdatedHandler(events.ScoreUpdated)
}

//Interface to HeExplode event subscribers
type HeExplodeSubscriber interface {
	HeExplodeHandler(events.HeExplode)
}

//Interface to ItemDrop event subscribers
type ItemDropSubscriber interface {
	ItemDropHandler(events.ItemDrop)
}

//Interface to ItemPickup event subscribers
type ItemPickupSubscriber interface {
	ItemPickupHandler(events.ItemPickup)
}

//Interface to Kill event subscribers
type KillSubscriber interface {
	KillHandler(events.Kill)
}

//Interface to PlayerFlashed event subscribers
type PlayerFlashedSubscriber interface {
	PlayerFlashedHandler(events.PlayerFlashed)
}

//Interface to PlayerHurt event subscribers
type PlayerHurtSubscriber interface {
	PlayerHurtHandler(events.PlayerHurt)
}

//Interface to WeaponReload event subscribers
type WeaponReloadSubscriber interface {
	WeaponReloadHandler(events.WeaponReload)
}

//Interface to IsWarmupPeriodChanged event subscribers
type IsWarmupPeriodChangedSubscriber interface {
	IsWarmupPeriodChangedHandler(events.IsWarmupPeriodChanged)
}

//Interface to PlayerTeamChange event subscribers
type PlayerTeamChangeSubscriber interface {
	PlayerTeamChangeHandler(events.PlayerTeamChange)
}

//basic shared handler for parsing
type basicHandler struct {
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
	isValidRound            bool
	terroristFirstTeamscore int
	ctFirstTeamScore        int
	playerMappings          []map[uint64]playerMapping
	matchDatetime           time.Time
}

func (bh *basicHandler) Register(basicHand *basicHandler) error {

	return nil
}

func (bh *basicHandler) Update() {

}

func (bh *basicHandler) RegisterBasicEvents() error {
	parser := *(bh.parser)
	bh.roundStartHandlerID = parser.RegisterEventHandler(bh.RoundStartHandler)
	bh.roundEndHandlerID = parser.RegisterEventHandler(bh.RoundEndHandler)
	bh.roundFreezeTimeEndHandlerID = parser.RegisterEventHandler(bh.RoundFreezetimeEndHandler)
	bh.playerTeamChangeHandlerID = parser.RegisterEventHandler(bh.PlayerTeamChangeHandler)
	return nil
}

func (bh *basicHandler) Setup(parser *dem.Parser, tickRate int, mapMetadata metadata.Map, matchDateTime time.Time) error {
	bh.parser = parser
	bh.tickRate = tickRate
	bh.mapMetadata = mapMetadata
	bh.statisticHolder.baseStatsHeaders = []string{"Rounds", "Rounds_T", "Rounds_CT"}
	bh.basicHandler = bh
	bh.matchDatetime = matchDateTime

	return nil
}

func (bh *basicHandler) UpdateTime() {
	bh.currentTime = getCurrentTime(*(bh.parser), bh.tickRate)
}

func (bh *basicHandler) GetPeriodicTabularData() ([]string, []float64, error) {
	parser := *(bh.parser)
	newCSVRow := []float64{0}
	currentRoundTime := getCurrentTime(parser, bh.tickRate)

	newCSVRow[0] = currentRoundTime - bh.roundStartTime
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

	currentMappings := currentPlayerMappings(parser.GameState())

	if len(currentMappings) > 0 && !bh.isMatchEnded {
		bh.isValidRound = true
	} else {
		bh.isValidRound = false
	}
	if bh.isValidRound {
		tTeam := gs.TeamTerrorists()
		ctTeam := gs.TeamCounterTerrorists()

		scoreDiff := utils.Abs((tTeam.Score() - ctTeam.Score()))
		isTMatchPoint := (tTeam.Score() >= 15 && tTeam.Score()%3 == 0 && scoreDiff >= 1)
		isCTMatchPoint := (ctTeam.Score() >= 15 && ctTeam.Score()%3 == 0 && scoreDiff >= 1)
		if bh.roundNumber == 29 {
			fmt.Println("A")
		}
		if isTMatchPoint || isCTMatchPoint {
			bh.matchPointTeam = bh.roundWinner
		} else {
			bh.matchPointTeam = ""
		}

		bh.roundFreezeTime = true
		bh.roundWinner = ""
		bh.frameGroup = 0
		bh.isMatchStarted = true
		bh.roundNumber = gs.TeamCounterTerrorists().Score() + gs.TeamTerrorists().Score() + 1
		bh.currentScore = utils.PadLeft(strconv.Itoa(bh.roundNumber), "0", 2) + "_ct_" +
			utils.PadLeft(strconv.Itoa(gs.TeamCounterTerrorists().Score()), "0", 2) +
			"_t_" + utils.PadLeft(strconv.Itoa(gs.TeamTerrorists().Score()), "0", 2)

		bh.roundStartTime = bh.currentTime

		if !bh.isMatchEnded && bh.isMatchStarted {
			if bh.roundNumber-1 < len(bh.playerMappings) {
				bh.CropData(bh.roundNumber - 1)
			}

			bh.playerMappings = append(bh.playerMappings, currentMappings)

			for _, subscriber := range bh.roundStartSubscribers {
				subscriber.RoundStartHandler(e)
			}
		}
	}

}

func (bh *basicHandler) getPlayersAlive(team common.Team) (playersAlive []*common.Player) {
	for _, playerMapping := range bh.playerMappings[bh.roundNumber-1] {
		player := playerMapping.playerObject
		if player.Team == team && player.IsAlive() {
			playersAlive = append(playersAlive, player)
		}
	}
	return playersAlive
}

func (bh *basicHandler) CropData(index int) {
	bh.playerMappings = bh.playerMappings[:index]
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
	tPoint := 0
	ctPoint := 0
	if winTeam == 2 {
		bh.roundWinner = "t"
		tPoint += 1
	} else if winTeam == 3 {
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

	if bh.isValidRound && bh.isMatchStarted {
		for _, subscriber := range bh.roundEndSubscribers {
			subscriber.RoundEndHandler(e)
		}
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
	if !bh.isMatchEnded && bh.isMatchStarted {
		for _, subscriber := range bh.grenadeEventIfSubscribers {
			subscriber.GrenadeEventIfHandler(e)
		}
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
	if !bh.isMatchEnded && bh.isMatchStarted {
		if bh.roundNumber-1 < len(bh.playerStats) {
			bh.playerStats = bh.playerStats[:bh.roundNumber-1]
		}
		bh.AddNewRound() //adds new round for statistic holder

		for _, player := range bh.playerMappings[bh.roundNumber-1] {
			bh.statisticHolder.setPlayerStat(player.playerObject, 1, "Rounds")
		}

		for _, subscriber := range bh.roundFreezeTimeEndSubscribers {
			subscriber.RoundFreezetimeEndHandler(e)
		}
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
	if !bh.isMatchEnded && bh.isMatchStarted {
		for _, subscriber := range bh.bombPlantedSubscribers {
			subscriber.BombPlantedHandler(e)
		}
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
	if !bh.isMatchEnded && bh.isMatchStarted {
		for _, subscriber := range bh.frameDoneSubscribers {
			subscriber.FrameDoneHandler(e)
		}
	}
}

func (bh *basicHandler) RegisterRoundEndOfficialSubscriber(rs RoundEndOfficialSubscriber) {
	parser := *(bh.parser)
	if bh.roundEndOfficialHandlerID == nil {
		bh.roundEndOfficialHandlerID = parser.RegisterEventHandler(bh.RoundEndOfficialHandler)
	}

	bh.roundEndOfficialSubscribers = append(bh.roundEndOfficialSubscribers, rs)

}

func (bh *basicHandler) RoundEndOfficialHandler(e events.RoundEndOfficial) {
	bh.UpdateTime()

	if bh.isMatchStarted && bh.isValidRound {
		for _, subscriber := range bh.roundEndOfficialSubscribers {
			subscriber.RoundEndOfficialHandler(e)
		}
	}
}

func (bh *basicHandler) RegisterBombDroppedSubscriber(rs BombDroppedSubscriber) {
	parser := *(bh.parser)
	if bh.bombDroppedHandlerID == nil {
		bh.bombDroppedHandlerID = parser.RegisterEventHandler(bh.BombDroppedHandler)
	}

	bh.bombDroppedSubscribers = append(bh.bombDroppedSubscribers, rs)

}

func (bh *basicHandler) BombDroppedHandler(e events.BombDropped) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted {
		for _, subscriber := range bh.bombDroppedSubscribers {
			subscriber.BombDroppedHandler(e)
		}
	}
}

func (bh *basicHandler) RegisterBombDefusedSubscriber(rs BombDefusedSubscriber) {
	parser := *(bh.parser)
	if bh.bombDefusedHandlerID == nil {
		bh.bombDefusedHandlerID = parser.RegisterEventHandler(bh.BombDefusedHandler)
	}

	bh.bombDefusedSubscribers = append(bh.bombDefusedSubscribers, rs)

}

func (bh *basicHandler) BombDefusedHandler(e events.BombDefused) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted {
		for _, subscriber := range bh.bombDefusedSubscribers {
			subscriber.BombDefusedHandler(e)
		}
	}
}

func (bh *basicHandler) RegisterFlashExplodeSubscriber(rs FlashExplodeSubscriber) {
	parser := *(bh.parser)
	if bh.flashExplodeHandlerID == nil {
		bh.flashExplodeHandlerID = parser.RegisterEventHandler(bh.FlashExplodeHandler)
	}

	bh.flashExplodeSubscribers = append(bh.flashExplodeSubscribers, rs)

}

func (bh *basicHandler) FlashExplodeHandler(e events.FlashExplode) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted {
		for _, subscriber := range bh.flashExplodeSubscribers {
			subscriber.FlashExplodeHandler(e)
		}
	}
}

func (bh *basicHandler) RegisterBombPickupSubscriber(rs BombPickupSubscriber) {
	parser := *(bh.parser)
	if bh.bombPickupHandlerID == nil {
		bh.bombPickupHandlerID = parser.RegisterEventHandler(bh.BombPickupHandler)
	}

	bh.bombPickupSubscribers = append(bh.bombPickupSubscribers, rs)

}

func (bh *basicHandler) BombPickupHandler(e events.BombPickup) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted {
		for _, subscriber := range bh.bombPickupSubscribers {
			subscriber.BombPickupHandler(e)
		}
	}
}

func (bh *basicHandler) RegisterFootstepSubscriber(rs FootstepSubscriber) {
	parser := *(bh.parser)
	if bh.footstepHandlerID == nil {
		bh.footstepHandlerID = parser.RegisterEventHandler(bh.FootstepHandler)
	}

	bh.footstepSubscribers = append(bh.footstepSubscribers, rs)

}

func (bh *basicHandler) FootstepHandler(e events.Footstep) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted {
		for _, subscriber := range bh.footstepSubscribers {
			subscriber.FootstepHandler(e)
		}
	}
}

func (bh *basicHandler) RegisterScoreUpdatedSubscriber(rs ScoreUpdatedSubscriber) {
	parser := *(bh.parser)
	if bh.scoreUpdatedHandlerID == nil {
		bh.scoreUpdatedHandlerID = parser.RegisterEventHandler(bh.ScoreUpdatedHandler)
	}

	bh.scoreUpdatedSubscribers = append(bh.scoreUpdatedSubscribers, rs)

}

func (bh *basicHandler) ScoreUpdatedHandler(e events.ScoreUpdated) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted {
		for _, subscriber := range bh.scoreUpdatedSubscribers {
			subscriber.ScoreUpdatedHandler(e)
		}
	}
}

func (bh *basicHandler) RegisterHeExplodeSubscriber(rs HeExplodeSubscriber) {
	parser := *(bh.parser)
	if bh.heExplodeHandlerID == nil {
		bh.heExplodeHandlerID = parser.RegisterEventHandler(bh.HeExplodeHandler)
	}

	bh.heExplodeSubscribers = append(bh.heExplodeSubscribers, rs)

}

func (bh *basicHandler) HeExplodeHandler(e events.HeExplode) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted {
		for _, subscriber := range bh.heExplodeSubscribers {
			subscriber.HeExplodeHandler(e)
		}
	}
}

func (bh *basicHandler) RegisterItemDropSubscriber(rs ItemDropSubscriber) {
	parser := *(bh.parser)
	if bh.itemDropHandlerID == nil {
		bh.itemDropHandlerID = parser.RegisterEventHandler(bh.ItemDropHandler)
	}

	bh.itemDropSubscribers = append(bh.itemDropSubscribers, rs)

}

func (bh *basicHandler) ItemDropHandler(e events.ItemDrop) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted {
		for _, subscriber := range bh.itemDropSubscribers {
			subscriber.ItemDropHandler(e)
		}
	}
}

func (bh *basicHandler) RegisterItemPickupSubscriber(rs ItemPickupSubscriber) {
	parser := *(bh.parser)
	if bh.itemPickupHandlerID == nil {
		bh.itemPickupHandlerID = parser.RegisterEventHandler(bh.ItemPickupHandler)
	}

	bh.itemPickupSubscribers = append(bh.itemPickupSubscribers, rs)

}

func (bh *basicHandler) ItemPickupHandler(e events.ItemPickup) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted {
		for _, subscriber := range bh.itemPickupSubscribers {
			subscriber.ItemPickupHandler(e)
		}
	}
}

func (bh *basicHandler) RegisterKillSubscriber(rs KillSubscriber) {
	parser := *(bh.parser)
	if bh.killHandlerID == nil {
		bh.killHandlerID = parser.RegisterEventHandler(bh.KillHandler)
	}

	bh.killSubscribers = append(bh.killSubscribers, rs)

}

func (bh *basicHandler) KillHandler(e events.Kill) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted {
		for _, subscriber := range bh.killSubscribers {
			subscriber.KillHandler(e)
		}
	}
}

func (bh *basicHandler) RegisterPlayerFlashedSubscriber(rs PlayerFlashedSubscriber) {
	parser := *(bh.parser)
	if bh.playerFlashedHandlerID == nil {
		bh.playerFlashedHandlerID = parser.RegisterEventHandler(bh.PlayerFlashedHandler)
	}

	bh.playerFlashedSubscribers = append(bh.playerFlashedSubscribers, rs)

}

func (bh *basicHandler) PlayerFlashedHandler(e events.PlayerFlashed) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted {
		for _, subscriber := range bh.playerFlashedSubscribers {
			subscriber.PlayerFlashedHandler(e)
		}
	}
}

func (bh *basicHandler) RegisterPlayerHurtSubscriber(rs PlayerHurtSubscriber) {
	parser := *(bh.parser)
	if bh.playerHurtHandlerID == nil {
		bh.playerHurtHandlerID = parser.RegisterEventHandler(bh.PlayerHurtHandler)
	}

	bh.playerHurtSubscribers = append(bh.playerHurtSubscribers, rs)

}

func (bh *basicHandler) PlayerHurtHandler(e events.PlayerHurt) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted {
		for _, subscriber := range bh.playerHurtSubscribers {
			subscriber.PlayerHurtHandler(e)
		}
	}
}

func (bh *basicHandler) RegisterWeaponReloadSubscriber(rs WeaponReloadSubscriber) {
	parser := *(bh.parser)
	if bh.weaponReloadHandlerID == nil {
		bh.weaponReloadHandlerID = parser.RegisterEventHandler(bh.WeaponReloadHandler)
	}

	bh.weaponReloadSubscribers = append(bh.weaponReloadSubscribers, rs)

}

func (bh *basicHandler) WeaponReloadHandler(e events.WeaponReload) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted {
		for _, subscriber := range bh.weaponReloadSubscribers {
			subscriber.WeaponReloadHandler(e)
		}
	}
}

func (bh *basicHandler) RegisterIsWarmupPeriodChangedSubscriber(rs IsWarmupPeriodChangedSubscriber) {
	parser := *(bh.parser)
	if bh.isWarmupPeriodChangedHandlerID == nil {
		bh.isWarmupPeriodChangedHandlerID = parser.RegisterEventHandler(bh.IsWarmupPeriodChangedHandler)
	}

	bh.isWarmupPeriodChangedSubscribers = append(bh.isWarmupPeriodChangedSubscribers, rs)

}

func (bh *basicHandler) IsWarmupPeriodChangedHandler(e events.IsWarmupPeriodChanged) {
	bh.UpdateTime()
	if !bh.isMatchEnded && bh.isMatchStarted {
		for _, subscriber := range bh.isWarmupPeriodChangedSubscribers {
			subscriber.IsWarmupPeriodChangedHandler(e)
		}
	}
}

func (bh *basicHandler) RegisterPlayerTeamChangeSubscriber(rs PlayerTeamChangeSubscriber) {
	parser := *(bh.parser)
	if bh.playerTeamChangeHandlerID == nil {
		bh.playerTeamChangeHandlerID = parser.RegisterEventHandler(bh.PlayerTeamChangeHandler)
	}

	bh.playerTeamChangeSubscribers = append(bh.playerTeamChangeSubscribers, rs)

}

func (bh *basicHandler) PlayerTeamChangeHandler(e events.PlayerTeamChange) {
	bh.UpdateTime()

	if !bh.isMatchEnded && bh.isMatchStarted {

		if e.NewTeam == common.TeamCounterTerrorists || e.NewTeam == common.TeamTerrorists {
			if _, ok := bh.playerMappings[bh.roundNumber-1][e.Player.SteamID64]; !ok {
				bh.playerMappings[bh.roundNumber-1] = currentPlayerMappings((*bh.parser).GameState())
			}
		}
		for _, subscriber := range bh.playerTeamChangeSubscribers {
			subscriber.PlayerTeamChangeHandler(e)
		}
	}
}

type poppingGrenadeHandler struct {
	basicHandler   *basicHandler
	activeGrenades []*grenadeTracker
	baseIcons      map[common.EquipmentType]utils.Icon
}

func (ph *poppingGrenadeHandler) Update() {

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

func (ph *poppingGrenadeHandler) GetPeriodicIcons() ([]utils.Icon, error) {
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

type playerPeriodicInfoHandler struct {
	basicHandler                *basicHandler
	periodicTabularInfoGatherer []IPlayersPeriodicTabularInfoGatherer
	periodicPlayerIconGatherer  []IPeriodicPlayerIconGatherer
}

func (ph *playerPeriodicInfoHandler) Register(bh *basicHandler) error {
	ph.basicHandler = bh

	bg := new(basicPlayerPositionGatherer)
	bg.Setup(bh)
	ph.periodicPlayerIconGatherer = append(ph.periodicPlayerIconGatherer, bg)
	ph.periodicTabularInfoGatherer = append(ph.periodicTabularInfoGatherer, new(hpGatherer))
	ph.periodicTabularInfoGatherer = append(ph.periodicTabularInfoGatherer, new(currentFlashTimeGatherer))
	ph.periodicTabularInfoGatherer = append(ph.periodicTabularInfoGatherer, new(weaponsGatherer))
	return nil
}

func (ph *playerPeriodicInfoHandler) Update() {
	var periodicGatherers []IPeriodicPlayerInfoGatherer
	for _, iconGatherer := range ph.periodicPlayerIconGatherer {
		iconGatherer.Init()
		periodicGatherers = append(periodicGatherers, iconGatherer)
	}
	for _, tabularGatherer := range ph.periodicTabularInfoGatherer {
		tabularGatherer.Init()
		periodicGatherers = append(periodicGatherers, tabularGatherer)
	}

	ph.updatePlayerInfo(periodicGatherers)

}

func (ph *playerPeriodicInfoHandler) updatePlayerInfo(playerInfoGatherers []IPeriodicPlayerInfoGatherer) {

	for _, playerMapping := range ph.basicHandler.playerMappings[ph.basicHandler.roundNumber-1] {
		player := playerMapping.playerObject

		for _, playerGatherer := range playerInfoGatherers {
			playerGatherer.updatePlayer(player, playerMapping.currentSlot)
		}

	}

}

func (ph *playerPeriodicInfoHandler) GetPeriodicTabularData() (newHeader []string, newCSVRow []float64, err error) {

	for _, periodicTabularGatherer := range ph.periodicTabularInfoGatherer {

		tempHeader, tempCSV := periodicTabularGatherer.GetPeriodicTabularInfo()
		newCSVRow = append(newCSVRow, tempCSV...)
		newHeader = append(newHeader, tempHeader...)
	}

	return newHeader, newCSVRow, err
}

func (ph *playerPeriodicInfoHandler) GetPeriodicIcons() ([]utils.Icon, error) {
	var iconList []utils.Icon
	for _, periodicIconGatherer := range ph.periodicPlayerIconGatherer {

		iconList = append(iconList, periodicIconGatherer.GetPlayerIcons()...)

	}
	return iconList, nil

}

type IPlayersInfoGatherer interface {
	Init()
}

type IPeriodicPlayerInfoGatherer interface {
	IPlayersInfoGatherer
	updatePlayer(*common.Player, int)
}

type IPeriodicPlayerIconGatherer interface {
	IPeriodicPlayerInfoGatherer
	GetPlayerIcons() []utils.Icon
}

type IPlayersPeriodicTabularInfoGatherer interface {
	IPeriodicPlayerInfoGatherer
	GetPeriodicTabularInfo() ([]string, []float64) //header,data
}

type playersTabularInfoGatherer struct {
	sizePerPlayer  int
	header         []string
	playersTabInfo []float64
}

type hpGatherer struct {
	playersInfoGatherer playersTabularInfoGatherer
}

func (hg *hpGatherer) Init() {
	hg.playersInfoGatherer.header = []string{"t_1", "t_2", "t_3", "t_4", "t_5", "ct_1", "ct_2", "ct_3", "ct_4", "ct_5"}

	hg.playersInfoGatherer.sizePerPlayer = len(hg.playersInfoGatherer.header) / 10
	hg.playersInfoGatherer.playersTabInfo = nil
	for range hg.playersInfoGatherer.header {
		hg.playersInfoGatherer.playersTabInfo = append(hg.playersInfoGatherer.playersTabInfo, 0.0)
	}

}

func (hg *hpGatherer) updatePlayer(player *common.Player, basePos int) {
	hg.playersInfoGatherer.playersTabInfo[basePos] = float64(player.Health()) / 100

}

func (hg *hpGatherer) GetPeriodicTabularInfo() ([]string, []float64) {
	return hg.playersInfoGatherer.header, hg.playersInfoGatherer.playersTabInfo

}

type currentFlashTimeGatherer struct {
	playersInfoGatherer playersTabularInfoGatherer
}

func (hg *currentFlashTimeGatherer) Init() {
	hg.playersInfoGatherer.header = []string{"t_1_blindtime", "t_2_blindtime", "t_3_blindtime", "t_4_blindtime", "t_5_blindtime",
		"ct_1_blindtime", "ct_2_blindtime", "ct_3_blindtime", "ct_4_blindtime", "ct_5_blindtime"}

	hg.playersInfoGatherer.sizePerPlayer = len(hg.playersInfoGatherer.header) / 10
	hg.playersInfoGatherer.playersTabInfo = nil
	for range hg.playersInfoGatherer.header {
		hg.playersInfoGatherer.playersTabInfo = append(hg.playersInfoGatherer.playersTabInfo, 0.0)
	}

}

func (hg *currentFlashTimeGatherer) updatePlayer(player *common.Player, basePos int) {

	hg.playersInfoGatherer.playersTabInfo[basePos] = player.FlashDurationTimeRemaining().Seconds()

}

func (hg *currentFlashTimeGatherer) GetPeriodicTabularInfo() ([]string, []float64) {
	return hg.playersInfoGatherer.header, hg.playersInfoGatherer.playersTabInfo

}

type weaponsGatherer struct {
	playersInfoGatherer playersTabularInfoGatherer
}

func (wg *weaponsGatherer) Init() {
	wg.playersInfoGatherer.header = []string{
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

	wg.playersInfoGatherer.sizePerPlayer = len(wg.playersInfoGatherer.header) / 10
	wg.playersInfoGatherer.playersTabInfo = nil
	for range wg.playersInfoGatherer.header {
		wg.playersInfoGatherer.playersTabInfo = append(wg.playersInfoGatherer.playersTabInfo, 0)
	}

}

func (wg *weaponsGatherer) updatePlayer(player *common.Player, basePos int) {
	//"mainweapon", "secweapon", "flashbangs", "hassmoke", "hasmolotov", "hashe","armorvalue","hashelmet","hasdefusekit/hasc4",

	weapons := []float64{0, 0, 0, 0, 0, 0, 0, 0, 0}

	primaryWeaponClasses := []int{2, 3, 4}
	secondaryWeaponClasses := []int{1}

	molotovAndIncendiary := []int{502, 503}

	equipSlice := player.Weapons()
	equipClass := 0
	equipType := 0
	for _, equip := range equipSlice {
		equipClass = int(equip.Class())
		equipType = int(equip.Type)
		if utils.FindIntInSlice(primaryWeaponClasses, equipClass) {
			weapons[0] = float64(equipType)
		}
		if utils.FindIntInSlice(secondaryWeaponClasses, equipClass) {
			weapons[1] = float64(equipType)
		}
		if equipType == 504 { //flash
			weapons[2] = float64(player.AmmoLeft[equip.AmmoType()])
		}
		if equipType == 505 { //smoke
			weapons[3] = 1
		}
		if utils.FindIntInSlice(molotovAndIncendiary, equipType) { //molotov or incendiary
			weapons[4] = 1
		}
		if equipType == 506 { //HE
			weapons[5] = 1
		}
		if equipType == 406 || player.HasDefuseKit() { //defuse kit / c4
			weapons[8] = 1
		}

	}
	weapons[6] = float64(player.Armor())
	if player.HasHelmet() {
		weapons[7] = 1
	}

	for i, weapon := range weapons {
		wg.playersInfoGatherer.playersTabInfo[basePos*wg.playersInfoGatherer.sizePerPlayer+i] = weapon
	}
}

func (wg *weaponsGatherer) GetPeriodicTabularInfo() ([]string, []float64) {
	return wg.playersInfoGatherer.header, wg.playersInfoGatherer.playersTabInfo

}

type statisticHolder struct {
	basicHandler        *basicHandler
	baseStatsHeaders    []string
	playerStats         []map[uint64][]float64
	defaultValues       map[string]float64
	ratioStats          [][3]string
	consolidatedHeaders []string
	consolidatedStats   map[uint64][]float64
}

func (kc statisticHolder) GetRoundStatistic(roundNumber int, userID uint64) ([]string, []float64, error) {

	return kc.baseStatsHeaders, kc.playerStats[roundNumber-1][userID], nil
}

func (kc *statisticHolder) addToPlayerStat(player *common.Player, addAmount float64, stat string) {
	isCT := (player.Team == 3)
	var suffix string
	kc.playerStats[kc.basicHandler.roundNumber-1][player.SteamID64][utils.IndexOf(stat, kc.baseStatsHeaders)] += addAmount
	if isCT {
		suffix = "_CT"
	} else {
		suffix = "_T"
	}
	if utils.IndexOf(stat+suffix, kc.baseStatsHeaders) != -1 {
		kc.playerStats[kc.basicHandler.roundNumber-1][player.SteamID64][utils.IndexOf(stat+suffix, kc.baseStatsHeaders)] += addAmount
	}
}

func (kc *statisticHolder) setPlayerStat(player *common.Player, value float64, stat string) {
	isCT := (player.Team == 3)
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
	for _, playerMapping := range kc.basicHandler.playerMappings[kc.basicHandler.roundNumber-1] {
		for _, header := range kc.baseStatsHeaders {
			if val, ok := kc.defaultValues[header]; ok {
				newStats = append(newStats, val)
			} else {
				newStats = append(newStats, 0)
			}

		}
		kc.playerStats[len(kc.playerStats)-1][playerMapping.playerObject.SteamID64] = newStats
	}

}

type playerConsolidatedStats struct {
	statsHeaders  []string
	playerName    string
	playerSteamID uint64
	playerStats   []float64
}

type matchConsolidatedStats struct {
	statsHeaders   []string
	playerNames    []string
	playerSteamIDs []uint64
	playerStats    [][]float64
}

type multiMatchConsolidatedStats struct {
	statsHeaders []string
	playerStats  map[uint64]playerConsolidatedStats
}

// func (kc *statisticHolder) ConsolidateMultimatchBaseStatistics(multiMatchStats []matchConsolidatedStats) (multiMatchConsolidation multiMatchConsolidatedStats) {
// 	multiMatchConsolidation = multiMatchConsolidatedStats{}

// 	var playerStat playerConsolidatedStats

// 	var statMapping map[string]int
// 	statMapping = make(map[string]int)

// 	copy(multiMatchConsolidation.statsHeaders, kc.baseStatsHeaders)

// 	for _, statName := range multiMatchConsolidation.statsHeaders {
// 		statMapping[statName] = utils.IndexOf(statName, multiMatchStats[0].statsHeaders)
// 	}

// 	for _, matchStats := range multiMatchStats { //consolidate base stats for all matches and players
// 		for playerIndex, playerSteamID := range matchStats.playerSteamIDs {
// 			if _, ok := multiMatchConsolidation.playerStats[playerSteamID]; !ok {
// 				playerStat = playerConsolidatedStats{}
// 				copy(playerStat.statsHeaders, multiMatchConsolidation.statsHeaders)
// 				playerStat.playerName = matchStats.playerNames[playerIndex]
// 				playerStat.playerSteamID = matchStats.playerSteamIDs[playerIndex]
// 				playerStat.playerStats = make([]float64, len(playerStat.statsHeaders))
// 			} else {
// 				playerStat = multiMatchConsolidation.playerStats[playerSteamID]
// 			}
// 			for statIndex, statName := range kc.baseStatsHeaders {
// 				playerStat.playerStats[statIndex] += matchStats.playerStats[playerIndex][statMapping[statName]]
// 			}
// 		}
// 	}
// 	return multiMatchConsolidation
// }

func (kc *statisticHolder) GetRatioStatistics() [][3]string {
	return kc.basicHandler.ratioStats
}

type PlayerStatisticCalculator interface {
	CompositeEventHandler
	GetRoundStatistic(roundNumber int, userID uint64) ([]string, []float64, error) //stats header, stats
	GetMatchStatistic(userID uint64) ([]string, []float64, error)                  //stats header, stats
}

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

func (kc *KDATCalculator) Register(bh *basicHandler) error {
	kc.basicHandler = bh
	bh.RegisterRoundStartSubscriber(interface{}(kc).(RoundStartSubscriber))
	bh.RegisterKillSubscriber(interface{}(kc).(KillSubscriber))
	bh.RegisterRoundEndOfficialSubscriber(interface{}(kc).(RoundEndOfficialSubscriber))
	bh.RegisterRoundEndSubscriber(interface{}(kc).(RoundEndSubscriber))
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

	kc.ratioStats = append(kc.ratioStats, [3]string{"KPR", "Kills", "Rounds"})
	kc.ratioStats = append(kc.ratioStats, [3]string{"APR", "Assists", "Rounds"})
	kc.ratioStats = append(kc.ratioStats, [3]string{"DPR", "Deaths", "Rounds"})
	kc.ratioStats = append(kc.ratioStats, [3]string{"Multikills Per Round", "Multikills", "Rounds"})
	kc.ratioStats = append(kc.ratioStats, [3]string{"First Kill success Percentage", "First Kills", "First Kill Attempts"})
	kc.ratioStats = append(kc.ratioStats, [3]string{"First Kill AttemptsPercentage", "First Kill Attempts", "Rounds"})
	kc.ratioStats = append(kc.ratioStats, [3]string{"Clutch Attempts Percentage", "Clutch Attempts", "Rounds"})
	kc.ratioStats = append(kc.ratioStats, [3]string{"Clutches Won Percentage", "Clutches", "Clutch Attempts"})
	kc.ratioStats = append(kc.ratioStats, [3]string{"HS Percentage", "HS Kills", "Rounds"})
	kc.ratioStats = append(kc.ratioStats, [3]string{"KAST", "KAST Sum", "Rounds"})
	kc.defaultValues = make(map[string]float64)
	return nil
}

func (kc *KDATCalculator) RoundStartHandler(e events.RoundStart) {
	if kc.basicHandler.roundNumber-1 < len(kc.playerStats) {
		kc.playerStats = kc.playerStats[:kc.basicHandler.roundNumber-1]
	}

	kc.killsToBeTraded = make(map[uint64][]KillToBeTraded)
	kc.isFirstDuel = true
	kc.clutchSituations = nil

}

func (kc *KDATCalculator) RoundFreezetimeEndHandler(e events.RoundFreezetimeEnd) {
	kc.AddNewRound()
}

func (kc *KDATCalculator) processClutchSituation(winnerTeam common.Team) {

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

func (kc *KDATCalculator) RoundEndHandler(e events.RoundEnd) {
	kc.processClutchSituation(e.Winner)

}

func (kc *KDATCalculator) RoundEndOfficialHandler(e events.RoundEndOfficial) {
	var playerKills float64
	var playerAssists float64
	var playerDeath float64
	var playerWasTraded float64
	var stringKills string
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

func (kc *KDATCalculator) addKDAInfo(e events.Kill) {
	if e.Killer != nil {
		kc.addToPlayerStat(e.Killer, 1, "Kills")
		if e.IsHeadshot {
			kc.addToPlayerStat(e.Killer, 1, "HS Kills")
		}
	}

	if e.Assister != nil {
		kc.addToPlayerStat(e.Assister, 1, "Assists")
	}

	if e.Victim != nil {
		kc.addToPlayerStat(e.Victim, 1, "Deaths")
		kc.addDeath(e.Victim)
	}

}

type clutchSituation struct {
	clutcher  *common.Player
	opponents []*common.Player
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
	if kc.isFirstDuel {
		if e.Killer != nil {
			kc.setPlayerStat(e.Killer, 1, "First Kills")
			kc.setPlayerStat(e.Killer, 1, "First Kill Attempts")
		}
		if e.Victim != nil {
			kc.setPlayerStat(e.Victim, 1, "First Kill Attempts")
		}

		kc.isFirstDuel = false
	}

}

func (kc *KDATCalculator) KillHandler(e events.Kill) {

	kc.addKDAInfo(e)
	kc.addFirstDuelInfo(e)

	kc.processKillTradeInformation(e)

}

type ADRCalculator struct {
	statisticHolder
}

func (kc *ADRCalculator) Register(bh *basicHandler) error {
	kc.basicHandler = bh
	bh.RegisterRoundStartSubscriber(interface{}(kc).(RoundStartSubscriber))
	bh.RegisterPlayerHurtSubscriber(interface{}(kc).(PlayerHurtSubscriber))
	bh.RegisterRoundFreezetimeEndSubscriber(interface{}(kc).(RoundFreezetimeEndSubscriber))
	kc.baseStatsHeaders = []string{"Total Damage Done", "Total Damage Done_T", "Total Damage Done_CT"}
	kc.ratioStats = append(kc.ratioStats, [3]string{"ADR", "Total Damage Done", "Rounds"})
	kc.defaultValues = make(map[string]float64)
	return nil
}

func (kc *ADRCalculator) RoundStartHandler(e events.RoundStart) {
	if kc.basicHandler.roundNumber-1 < len(kc.playerStats) {
		kc.playerStats = kc.playerStats[:kc.basicHandler.roundNumber-1]
	}
}

func (kc *ADRCalculator) RoundFreezetimeEndHandler(e events.RoundFreezetimeEnd) {

	kc.AddNewRound()
}

func (kc *ADRCalculator) PlayerHurtHandler(e events.PlayerHurt) {

	if e.Attacker != nil {
		kc.addToPlayerStat(e.Attacker, float64(e.HealthDamageTaken), "Total Damage Done")
	}

}

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

func (fc *FlashUsageCalculator) Register(bh *basicHandler) error {
	fc.basicHandler = bh
	bh.RegisterRoundStartSubscriber(interface{}(fc).(RoundStartSubscriber))
	bh.RegisterFlashExplodeSubscriber(interface{}(fc).(FlashExplodeSubscriber))
	bh.RegisterKillSubscriber(interface{}(fc).(KillSubscriber))
	bh.RegisterPlayerFlashedSubscriber(interface{}(fc).(PlayerFlashedSubscriber))
	bh.RegisterRoundEndOfficialSubscriber(interface{}(fc).(RoundEndOfficialSubscriber))
	bh.RegisterRoundFreezetimeEndSubscriber(interface{}(fc).(RoundFreezetimeEndSubscriber))
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

func (fc *FlashUsageCalculator) RoundEndOfficialHandler(e events.RoundEndOfficial) {
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

type playersIconGatherer struct {
	playersIcons []utils.Icon
	mapMetadata  metadata.Map
}

type basicPlayerPositionGatherer struct {
	playersIconGatherer playersIconGatherer
}

func (bg *basicPlayerPositionGatherer) Init() {
	bg.playersIconGatherer.playersIcons = nil
}

func (bg *basicPlayerPositionGatherer) Setup(basicHandler *basicHandler) {
	bg.playersIconGatherer.mapMetadata = basicHandler.mapMetadata
}

func (bg *basicPlayerPositionGatherer) updatePlayer(player *common.Player, basePos int) {

	if player.Health() > 0 {
		x, y := bg.playersIconGatherer.mapMetadata.TranslateScale(player.Position().X, player.Position().Y)
		var icon string

		if basePos/5 == 1 {

			icon = "ct_"
			if player.HasDefuseKit() {
				newIcon := utils.Icon{X: x, Y: y, IconName: "kit"} //t or ct icon
				bg.playersIconGatherer.playersIcons = append(bg.playersIconGatherer.playersIcons, newIcon)
			}

		} else {
			icon = "terrorist_"

		}
		playerNumber := basePos%5 + 1 //count 1-5 tr, 6-10 ct

		newIcon := utils.Icon{X: x, Y: y, IconName: icon + strconv.Itoa(playerNumber), Rotate: float64(player.ViewDirectionX())} //t or ct icon
		bg.playersIconGatherer.playersIcons = append(bg.playersIconGatherer.playersIcons, newIcon)
		newIcon = utils.Icon{X: x, Y: y, IconName: strconv.Itoa(playerNumber)}
		bg.playersIconGatherer.playersIcons = append(bg.playersIconGatherer.playersIcons, newIcon)
	}

}

func (bg *basicPlayerPositionGatherer) GetPlayerIcons() []utils.Icon {
	return bg.playersIconGatherer.playersIcons
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

func (bmbh *bombHandler) Update() {

}

func (bh *bombHandler) BombPlantedHandler(e events.BombPlanted) {
	parser := (*bh.basicHandler.parser)
	bh.bombPlanted = true
	bh.bombPlantedTime = getCurrentTime(parser, bh.basicHandler.tickRate)

}

func (bh *bombHandler) RoundStartHandler(e events.RoundStart) {

	bh.bombPlanted = false

}

func (bh *bombHandler) GetPeriodicIcons() (icons []utils.Icon, err error) {
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

func (bh *bombHandler) GetPeriodicTabularData() ([]string, []float64, error) {
	newCSVRow := []float64{0}
	if bh.bombPlanted {
		newCSVRow[0] = bh.basicHandler.currentTime - bh.bombPlantedTime
	}

	header := []string{"bomb_timeticking"}
	return header, newCSVRow, nil
}

type grenadeTracker struct {
	grenadeEvent events.GrenadeEvent
	grenadeTime  float64
}

type matchData struct {
	matchIcons                      [][][]utils.Icon //dimensions: rounds x frames x icons
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
	md.matchIcons = append(md.matchIcons, [][]utils.Icon{})
	md.matchPeriodicTabularData = append(md.matchPeriodicTabularData, [][]float64{})
	md.matchStatistics = append(md.matchStatistics, []float64{})
}

func (md *matchData) AddNewFrameGroup(roundNumber int) {
	if len(md.matchIcons[roundNumber]) == 0 {
		md.matchPeriodicTabularData[roundNumber] = append(md.matchPeriodicTabularData[roundNumber], []float64{})
	}
	md.matchIcons[roundNumber] = append(md.matchIcons[roundNumber], []utils.Icon{})
	md.matchPeriodicTabularData[roundNumber] = append(md.matchPeriodicTabularData[roundNumber], []float64{})

}

type infoGenerationHandler struct {
	basicHandler            *basicHandler
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
	mapGenerator            utils.MapGenerator
	matchData               *matchData
	imgSize                 int
	demFileHash             string

	rootMatchPath    string
	roundDirPath     string
	roundTabularPath string
	roundStatPath    string
	playerStatPath   string
}

func (ih *infoGenerationHandler) Register(bh *basicHandler) error {
	ih.basicHandler = bh

	bh.RegisterRoundStartSubscriber(interface{}(ih).(RoundStartSubscriber))
	bh.RegisterFrameDoneSubscriber(interface{}(ih).(FrameDoneSubscriber))
	bh.RegisterRoundEndOfficialSubscriber(interface{}(ih).(RoundEndOfficialSubscriber))

	return nil
}

func (ih *infoGenerationHandler) RoundStartHandler(e events.RoundStart) {
	if !ih.basicHandler.isMatchEnded {
		ih.roundDirPath = ih.rootMatchPath + "/" + ih.basicHandler.currentScore
		dirExists, _ := exists(ih.roundDirPath)
		ih.generationIndex = 0
		ih.roundEndRegistered = false

		if !dirExists {
			err := os.MkdirAll(ih.roundDirPath, 0700)
			checkError(err)
		} else {
			RemoveContents(ih.roundDirPath)
		}

		ih.roundTabularPath = ih.roundDirPath + "/periodic_data.csv"
		ih.roundStatPath = ih.roundDirPath + "/statistics.csv"
		ih.playerStatPath = ih.roundDirPath + "/player_statistics.csv"
		ih.isNewRound = true
		roundPeriodicDataCSV, err := os.Create(ih.roundTabularPath)
		checkError(err)
		defer roundPeriodicDataCSV.Close()

		roundStatCSV, err := os.Create(ih.roundStatPath)
		checkError(err)
		defer roundStatCSV.Close()

		playerStatCSV, err := os.Create(ih.playerStatPath)
		checkError(err)
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

func (ih *infoGenerationHandler) GetFullRoundStatistics() (data [][]string) {
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

			checkError(err)

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

func (ih *infoGenerationHandler) RoundEndOfficialHandler(e events.RoundEndOfficial) {

	if ih.basicHandler.roundWinner != "" && !ih.matchEndRegisted {
		fmt.Println("Generating round ", ih.basicHandler.roundNumber)

		var tempHeader []string
		var tempData []float64
		var newHeaderStat []string
		var newStat []float64
		var err error

		if ih.roundDirPath != ih.rootMatchPath {
			fileWrite, err := os.Create(ih.roundDirPath + "/winner.txt")
			checkError(err)
			defer fileWrite.Close()
			_, err = fileWrite.WriteString(ih.basicHandler.roundWinner)
			checkError(err)

		}

		for _, statGenerator := range *ih.allStatGenerators {
			newHeaderStat, newStat, err = statGenerator.GetStatistics()
			checkError(err)
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

		generateRoundMaps(ih.mapGenerator, ih.matchData.matchIcons[ih.basicHandler.roundNumber-1],
			ih.roundDirPath, ih.imgSize)
		writeToCSV(allTabularData, ih.roundTabularPath)
		writeToCSV(generalStatistics, ih.roundStatPath)
		writeToCSV(playerStatistics, ih.playerStatPath)

		ih.checkAndGenerateMatchEndStatistics()
		ih.roundEndRegistered = true
	}
}

func (ih *infoGenerationHandler) checkAndGenerateMatchEndStatistics() {

	if ih.basicHandler.isMatchEnded {
		fmt.Println("Generating match statistics")
		if ih.roundDirPath != ih.rootMatchPath {
			data := ih.GetFullMatchStatistics()
			fileWrite, err := os.Create(ih.rootMatchPath + "/match_statistics.csv")
			checkError(err)
			writer := csv.NewWriter(fileWrite)

			err = writer.WriteAll(data)
			checkError(err)
			defer fileWrite.Close()
			ih.matchEndRegisted = true
		}
	}

}

func (ih *infoGenerationHandler) GetFullMatchStatistics() (data [][]string) {
	var tempHeader []string
	var tempData []float64
	var err error
	var stringData []string
	var framedData [10][]string
	var statsIDs [][]int

	firstPlayer := true
	data = append(data, []string{"Name", "SteamID"})
	dbConn := openDBConn()

	matchID := insertMatch(dbConn, ih.demFileHash, ih.basicHandler.mapMetadata.Name,
		ih.basicHandler.terroristFirstTeamscore, ih.basicHandler.ctFirstTeamScore, ih.basicHandler.matchDatetime, true)

	for _, playerMapping := range ih.basicHandler.playerMappings[ih.basicHandler.roundNumber-1] {
		player := playerMapping.playerObject

		for j, playerStatCalculator := range *ih.allPlayerStatCalculator {
			tempHeader, tempData, err = playerStatCalculator.GetMatchStatistic(player.SteamID64)
			checkError(err)

			if j == 0 {
				insertPlayer(dbConn, playerMapping.playerObject.SteamID64, playerMapping.playerObject.Name)
				stringData = append([]string{playerMapping.playerObject.Name,
					strconv.FormatUint(playerMapping.playerObject.SteamID64, 10)}, utils.FloatSliceToString(tempData)...)
			} else {
				stringData = append(stringData, utils.FloatSliceToString(tempData)...)
			}

			if firstPlayer {
				data[0] = append(data[0], tempHeader...)
				statsIDs = append(statsIDs, insertBaseStatistics(dbConn, tempHeader))
			}
			insertStatisticsFacts(dbConn, statsIDs[j], tempData, player.SteamID64, matchID)

		}
		framedData[playerMapping.currentSlot] = append(framedData[playerMapping.currentSlot], stringData...)
		firstPlayer = false

	}
	data = append(data, framedData[:]...)
	return data

}

func (ih *infoGenerationHandler) FrameDoneHandler(e events.FrameDone) {

	if ih.isReadyForProcessing() {
		ih.processFrameEnd()
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

func (ih *infoGenerationHandler) Setup(imgSize int, updateInterval float64, rootMatchPath string, demFileHash string,
	allIconGenerators *[]PeriodicIconGenerator, allTabularGenerators *[]PeriodicTabularGenerator,
	allStatGenerators *[]StatGenerator, allPlayerStatCalculators *[]PlayerStatisticCalculator) error {

	var mapGenerator utils.MapGenerator
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

func (ih *infoGenerationHandler) processFrameEnd() {
	var newIcons []utils.Icon

	for _, iconGenerator := range *ih.allIconGenerators {
		iconGenerator.Update()
		tempIcons, err := iconGenerator.GetPeriodicIcons()
		checkError(err)
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
		checkError(err)
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
	ih.lastUpdate = getRoundTime(parser, ih.basicHandler.roundStartTime, ih.basicHandler.tickRate)
}

//var allPlayers = make(map[int]*playerMapping)

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
	fileStat, err := os.Stat(demPath)

	f, err := os.Open(demPath)
	checkError(err)
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Erro no processamento do arquivo!", r)
		}
	}()

	hasher := sha256.New()

	_, err = io.Copy(hasher, f)
	checkError(err)
	defer f.Close()

	f, err = os.Open(demPath)
	checkError(err)

	p := dem.NewParser(f)

	header, err := p.ParseHeader()
	checkError(err)

	fmt.Println("Map:", header.MapName)
	rootMatchPath := destDir + "/" + header.MapName + "/" + strconv.Itoa(fileID)
	dirExists, _ := exists(rootMatchPath)
	if !dirExists {
		err = os.MkdirAll(rootMatchPath, 0700)
		checkError(err)
	}

	imgSize := 300

	mapMetadata := metadata.MapNameToMap[header.MapName]
	var mapGenerator utils.MapGenerator
	var allIconGenerators []PeriodicIconGenerator
	var allStatGenerators []StatGenerator
	var allTabularGenerators []PeriodicTabularGenerator
	var allPlayerStatCalculators []PlayerStatisticCalculator
	var basicHandler basicHandler

	mapGenerator.Setup(header.MapName, imgSize)

	basicHandler.Setup(&p, tickRate, mapMetadata, fileStat.ModTime())
	basicHandler.RegisterBasicEvents()
	allTabularGenerators = append(allTabularGenerators, &basicHandler)
	allPlayerStatCalculators = append(allPlayerStatCalculators, &basicHandler)

	tradeIntervalLimit := 5.0
	var kdatHandler KDATCalculator
	kdatHandler.Register(&basicHandler)
	kdatHandler.Setup(tradeIntervalLimit)
	allPlayerStatCalculators = append(allPlayerStatCalculators, &kdatHandler)

	var adrHandler ADRCalculator
	adrHandler.Register(&basicHandler)
	allPlayerStatCalculators = append(allPlayerStatCalculators, &adrHandler)

	var flashCalc FlashUsageCalculator
	flashCalc.Register(&basicHandler)
	allPlayerStatCalculators = append(allPlayerStatCalculators, &flashCalc)

	var popHandler poppingGrenadeHandler
	popHandler.SetBaseIcons()
	popHandler.Register(&basicHandler)
	//allIconGenerators = append(allIconGenerators, &popHandler)

	var bmbHandler bombHandler
	bmbHandler.Register(&basicHandler)
	allTabularGenerators = append(allTabularGenerators, &bmbHandler)
	//allIconGenerators = append(allIconGenerators, &bmbHandler)

	var playerHandler playerPeriodicInfoHandler
	playerHandler.Register(&basicHandler)
	allTabularGenerators = append(allTabularGenerators, &playerHandler)
	//allIconGenerators = append(allIconGenerators, &playerHandler)

	var infoHandler infoGenerationHandler
	updateInterval := 1.5 //1.5s between framegroups
	infoHandler.Register(&basicHandler)
	infoHandler.Setup(imgSize, updateInterval, rootMatchPath, hex.EncodeToString(hasher.Sum(nil)),
		&allIconGenerators, &allTabularGenerators, &allStatGenerators, &allPlayerStatCalculators)

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
	defer file.Close()
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

func sortPlayersByUserID(allPlayers map[uint64]playerMapping) []uint64 {

	var keys []uint64
	for userID := range allPlayers {
		keys = append(keys, userID)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
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

func insertPlayer(dbConn *sql.DB, playerID uint64, playerName string) {
	insForm, err := dbConn.Prepare("INSERT INTO PLAYER(idPLAYER, NAME) VALUES(?,?) ON DUPLICATE KEY UPDATE NAME=?")
	checkError(err)
	insForm.Exec(playerID, playerName, playerName)
	insForm.Close()
}

func insertMatch(dbConn *sql.DB, demFileHash string, mapName string, terroristFirstTeamScore int, ctFirstTeamScore int,
	matchDateTime time.Time, overwriteMatch bool) (matchID int) {

	dt := matchDateTime.Format(time.RFC3339)
	insForm, err := dbConn.Prepare("INSERT INTO CSGO_MATCH(SCORE_FIRST_T,SCORE_FIRST_CT,MAP,MATCH_DATETIME,DEMO_FILE_HASH) VALUES(?,?,?,?,?)")
	checkError(err)
	_, err = insForm.Exec(terroristFirstTeamScore, ctFirstTeamScore, mapName, dt, demFileHash)
	insForm.Close()
	if err.(*mysql.MySQLError).Number == 1062 {
		fmt.Println("Demo file already in database")
		if overwriteMatch {
			fmt.Println("Overwrite mode on. Deleting old data.")
			insForm, err = dbConn.Prepare("DELETE STATISTICS_PLAYER_MATCH_FACT FROM STATISTICS_PLAYER_MATCH_FACT INNER JOIN " +
				"CSGO_MATCH ON STATISTICS_PLAYER_MATCH_FACT.idCSGO_MATCH = CSGO_MATCH.idCSGO_MATCH WHERE CSGO_MATCH.DEMO_FILE_HASH = ?")
			_, err = insForm.Exec(demFileHash)
			insForm.Close()
			checkError(err)
		} else {
			checkError(err)
		}
	}

	sqlResult, err := dbConn.Query("SELECT idCSGO_MATCH FROM CSGO_MATCH WHERE DEMO_FILE_HASH=?", demFileHash)
	checkError(err)
	sqlResult.Next()
	sqlResult.Scan(&matchID)
	sqlResult.Close()
	return matchID
}

func openDBConn() *sql.DB {
	db, err := sql.Open("mysql", "marcel:basecsteste1!@tcp(127.0.0.1:3306)/CSGO_ANALYTICS")

	checkError(err)
	return db
}

func insertBaseStatistics(dbConn *sql.DB, tempHeader []string) (statIds []int) {
	var newID int
	for _, statName := range tempHeader {
		insForm, err := dbConn.Prepare("INSERT IGNORE INTO BASE_STATISTIC(NAME) VALUES(?)")
		checkError(err)
		insForm.Exec(statName)
		insForm.Close()
		sqlResult, err := dbConn.Query("SELECT idBASE_STATISTIC FROM BASE_STATISTIC WHERE NAME=?", statName)
		checkError(err)
		sqlResult.Next()
		sqlResult.Scan(&newID)
		sqlResult.Close()
		statIds = append(statIds, newID)

	}
	return statIds
}

func insertStatisticsFacts(dbConn *sql.DB, statIDs []int, tempData []float64, playerID uint64, matchID int) {
	for i, statID := range statIDs {
		insForm, err := dbConn.Prepare("INSERT INTO STATISTICS_PLAYER_MATCH_FACT(idCSGO_MATCH,idPLAYER,idBASE_STATISTIC,VALUE) VALUES(?,?,?,?)")
		checkError(err)
		insForm.Exec(matchID, playerID, statID, tempData[i])
		insForm.Close()
	}
}
