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

	roundStartTime  float64
	currentTime     float64
	currentScore    string
	roundNumber     int
	frameGroup      int
	isMatchStarted  bool
	roundFreezeTime bool
	roundWinner     string
	matchPointTeam  string
	isMatchEnded    bool
	isValidRound    bool
	playerMappings  []map[int]playerMapping
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
	return nil
}

func (bh *basicHandler) Setup(parser *dem.Parser, tickRate int, mapMetadata metadata.Map) error {
	bh.parser = parser
	bh.tickRate = tickRate
	bh.mapMetadata = mapMetadata

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

	if len(currentMappings) > 0 {
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

		if bh.roundNumber-1 < len(bh.playerMappings) {
			bh.CropData(bh.roundNumber - 1)
		}

		bh.playerMappings = append(bh.playerMappings, currentMappings)

		if !bh.isMatchEnded && bh.isMatchStarted {
			for _, subscriber := range bh.roundStartSubscribers {
				subscriber.RoundStartHandler(e)
			}
		}
	}

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
	if winTeam == 2 {
		bh.roundWinner = "t"
	} else if winTeam == 3 {
		bh.roundWinner = "ct"
	} else {
		bh.roundWinner = "invalid"
	}
	if !bh.isMatchEnded && bh.isMatchStarted {
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
	if bh.roundWinner == bh.matchPointTeam && bh.matchPointTeam != "" && bh.isMatchStarted {
		bh.isMatchEnded = true
	}

	if bh.isMatchStarted {
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
	statsHeaders  []string
	playerStats   []map[int][]float64
	defaultValues []float64
}

func (kc statisticHolder) GetRoundStatistic(roundNumber int, userID int) ([]string, []float64, error) {
	return kc.statsHeaders, kc.playerStats[roundNumber-1][userID], nil
}

type PlayerStatisticCalculator interface {
	CompositeEventHandler
	GetRoundStatistic(roundNumber int, userID int) ([]string, []float64, error) //stats header, stats
	GetMatchStatistic(userID int) ([]string, []float64, error)                  //stats header, stats
}

type KDATCalculator struct {
	basicHandler *basicHandler
	statisticHolder
	killsToBeTraded    map[int][]KillToBeTraded //maps from killerID to a list of their kills
	tradeIntervalLimit float64
}

type KillToBeTraded struct {
	killerID    int
	victimID    int
	timeOfDeath float64
}

func (kc *KDATCalculator) processKillTradeInformation(e events.Kill) {
	if e.Killer != nil {
		killerID := e.Killer.UserID
		victimID := e.Victim.UserID
		var timeFromKill float64
		var victimOfVictimID int
		currentTime := kc.basicHandler.currentTime
		if victimKills, ok := kc.killsToBeTraded[victimID]; ok {
			for _, victimKill := range victimKills {
				timeFromKill = currentTime - victimKill.timeOfDeath
				if timeFromKill < kc.tradeIntervalLimit {
					victimOfVictimID = victimKill.victimID
					kc.playerStats[kc.basicHandler.roundNumber-1][victimOfVictimID][utils.IndexOf("WasTraded", kc.statsHeaders)] += 1
					kc.playerStats[kc.basicHandler.roundNumber-1][killerID][utils.IndexOf("Trades", kc.statsHeaders)] += 1
				}
			}

		}
		kc.killsToBeTraded[killerID] = append(kc.killsToBeTraded[killerID],
			KillToBeTraded{killerID: killerID, victimID: victimID, timeOfDeath: currentTime})
	}

}

func (kc *KDATCalculator) addPlayerToBeTraded(killerID int, victimID int, timeOfDeath float64) {
	kc.killsToBeTraded[killerID] = append(kc.killsToBeTraded[killerID],
		KillToBeTraded{killerID: killerID, victimID: victimID, timeOfDeath: timeOfDeath})
}

func (kc *KDATCalculator) Setup(tradeIntervalLimit float64) {
	kc.tradeIntervalLimit = tradeIntervalLimit
}

func (kc *KDATCalculator) GetMatchStatistic(userID int) ([]string, []float64, error) {
	consolidatedStat := []float64{}
	for _, roundStatMap := range kc.playerStats {
		if playerStat, ok := roundStatMap[userID]; ok {

			consolidatedStat = utils.ElementWiseSum(consolidatedStat, playerStat)
		}
	}
	consolidatedStat[utils.IndexOf("KAST", kc.statsHeaders)] = consolidatedStat[utils.IndexOf("KAST", kc.statsHeaders)] / float64(len(kc.playerStats))
	return kc.statsHeaders, consolidatedStat, nil
}

func (kc *KDATCalculator) Register(bh *basicHandler) error {
	kc.basicHandler = bh
	bh.RegisterRoundStartSubscriber(interface{}(kc).(RoundStartSubscriber))
	bh.RegisterKillSubscriber(interface{}(kc).(KillSubscriber))
	bh.RegisterRoundEndOfficialSubscriber(interface{}(kc).(RoundEndOfficialSubscriber))
	kc.statsHeaders = []string{"Kills", "Assists", "Deaths", "Trades", "WasTraded", "KAST"}
	kc.defaultValues = []float64{0, 0, 0, 0, 0, 0}
	return nil
}

func (kc *KDATCalculator) AddNewRound() {

	kc.playerStats = append(kc.playerStats, make(map[int][]float64))
	for _, playerMapping := range kc.basicHandler.playerMappings[kc.basicHandler.roundNumber-1] {
		for i, _ := range kc.statsHeaders {
			kc.playerStats[len(kc.playerStats)-1][playerMapping.playerObject.UserID] =
				append(kc.playerStats[len(kc.playerStats)-1][playerMapping.playerObject.UserID], kc.defaultValues[i])
		}

	}

}

func (kc *KDATCalculator) RoundStartHandler(e events.RoundStart) {
	if kc.basicHandler.roundNumber-1 < len(kc.playerStats) {
		kc.playerStats = kc.playerStats[:kc.basicHandler.roundNumber-1]
	}
	kc.killsToBeTraded = make(map[int][]KillToBeTraded)
	kc.AddNewRound()
}

func (kc *KDATCalculator) RoundEndOfficialHandler(e events.RoundEndOfficial) {
	var playerKills float64
	var playerAssists float64
	var playerDeath float64
	var playerWasTraded float64
	if kc.basicHandler.isMatchStarted {
		for _, playerMapping := range kc.basicHandler.playerMappings[kc.basicHandler.roundNumber-1] {
			playerKills = kc.playerStats[kc.basicHandler.roundNumber-1][playerMapping.playerObject.UserID][utils.IndexOf("Kills", kc.statsHeaders)]
			playerAssists = kc.playerStats[kc.basicHandler.roundNumber-1][playerMapping.playerObject.UserID][utils.IndexOf("Assists", kc.statsHeaders)]
			playerDeath = kc.playerStats[kc.basicHandler.roundNumber-1][playerMapping.playerObject.UserID][utils.IndexOf("Deaths", kc.statsHeaders)]
			playerWasTraded = kc.playerStats[kc.basicHandler.roundNumber-1][playerMapping.playerObject.UserID][utils.IndexOf("WasTraded", kc.statsHeaders)]

			if playerKills > 0 || playerAssists > 0 || playerDeath == 0 || playerWasTraded > 0 {
				kc.playerStats[kc.basicHandler.roundNumber-1][playerMapping.playerObject.UserID][utils.IndexOf("KAST", kc.statsHeaders)] = 1
			}

		}
	}

}

func (kc *KDATCalculator) KillHandler(e events.Kill) {
	if kc.basicHandler.isMatchStarted {

		if e.Killer != nil {
			kc.playerStats[kc.basicHandler.roundNumber-1][e.Killer.UserID][utils.IndexOf("Kills", kc.statsHeaders)] += 1 //add kill
		}

		if e.Assister != nil {
			kc.playerStats[kc.basicHandler.roundNumber-1][e.Assister.UserID][utils.IndexOf("Assists", kc.statsHeaders)] += 1 //add assist
		}

		if e.Victim != nil {
			kc.playerStats[kc.basicHandler.roundNumber-1][e.Victim.UserID][utils.IndexOf("Deaths", kc.statsHeaders)] += 1 //add death
		}

		kc.processKillTradeInformation(e)

	}

}

type ADRCalculator struct {
	basicHandler *basicHandler
	statisticHolder
}

func (kc *ADRCalculator) GetMatchStatistic(userID int) ([]string, []float64, error) {
	consolidatedStat := []float64{}
	for _, roundStatMap := range kc.playerStats {
		if playerStat, ok := roundStatMap[userID]; ok {

			consolidatedStat = utils.ElementWiseSum(consolidatedStat, playerStat)

		}

	}
	consolidatedStat = utils.ElementWiseDivision(consolidatedStat, float64(len(kc.playerStats)))
	return kc.statsHeaders, consolidatedStat, nil
}

func (kc *ADRCalculator) Register(bh *basicHandler) error {
	kc.basicHandler = bh
	bh.RegisterRoundStartSubscriber(interface{}(kc).(RoundStartSubscriber))
	bh.RegisterPlayerHurtSubscriber(interface{}(kc).(PlayerHurtSubscriber))
	kc.statsHeaders = []string{"ADR"}
	return nil
}

func (kc *ADRCalculator) AddNewRound() {

	kc.playerStats = append(kc.playerStats, make(map[int][]float64))
	for _, playerMapping := range kc.basicHandler.playerMappings[kc.basicHandler.roundNumber-1] {
		for range kc.statsHeaders {
			kc.playerStats[len(kc.playerStats)-1][playerMapping.playerObject.UserID] =
				append(kc.playerStats[len(kc.playerStats)-1][playerMapping.playerObject.UserID], 0)
		}

	}

}

func (kc *ADRCalculator) RoundStartHandler(e events.RoundStart) {
	if kc.basicHandler.roundNumber-1 < len(kc.playerStats) {
		kc.playerStats = kc.playerStats[:kc.basicHandler.roundNumber-1]
	}
	kc.AddNewRound()
}

func (kc *ADRCalculator) PlayerHurtHandler(e events.PlayerHurt) {

	if e.Attacker != nil {
		if _, ok := kc.playerStats[kc.basicHandler.roundNumber-1][e.Attacker.UserID]; !ok {
			fmt.Println("nokey")
		}
		kc.playerStats[kc.basicHandler.roundNumber-1][e.Attacker.UserID][0] += float64(e.HealthDamageTaken) //add damage
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
	bh.RegisterPlayerTeamChangeSubscriber(interface{}(ih).(PlayerTeamChangeSubscriber))
	bh.RegisterIsWarmupPeriodChangedSubscriber(interface{}(ih).(IsWarmupPeriodChangedSubscriber))

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
			tempHeader, tempData, err = playerStatCalculator.GetRoundStatistic(ih.basicHandler.roundNumber, player.UserID)

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
			fileWrite, err := os.Create(ih.rootMatchPath + "/match_statistics.txt")
			checkError(err)
			writer := csv.NewWriter(fileWrite)

			err = writer.WriteAll(data)
			checkError(err)
			defer fileWrite.Close()
			ih.matchEndRegisted = true
		}
	}

}

func (ih *infoGenerationHandler) PlayerTeamChangeHandler(e events.PlayerTeamChange) {
	if _, ok := ih.basicHandler.playerMappings[ih.basicHandler.roundNumber-1][e.Player.UserID]; !ok {
		fmt.Println("PlayerTeamChange")
	}

}

func (ih *infoGenerationHandler) IsWarmupPeriodChangedHandler(e events.IsWarmupPeriodChanged) {
	fmt.Println("WarmUpPeriodChanged")
}

func (ih *infoGenerationHandler) GetFullMatchStatistics() (data [][]string) {
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
			tempHeader, tempData, err = playerStatCalculator.GetMatchStatistic(player.UserID)

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

func (ih *infoGenerationHandler) Setup(imgSize int, updateInterval float64, rootMatchPath string, allIconGenerators *[]PeriodicIconGenerator, allTabularGenerators *[]PeriodicTabularGenerator,
	allStatGenerators *[]StatGenerator, allPlayerStatCalculators *[]PlayerStatisticCalculator) error {

	var mapGenerator utils.MapGenerator
	mapGenerator.Setup(ih.basicHandler.mapMetadata.Name, imgSize)
	ih.mapGenerator = mapGenerator
	ih.updateInterval = updateInterval
	ih.matchData = new(matchData)
	ih.rootMatchPath = rootMatchPath
	ih.roundDirPath = rootMatchPath

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

	imgSize := 500

	mapMetadata := metadata.MapNameToMap[header.MapName]
	var mapGenerator utils.MapGenerator
	var allIconGenerators []PeriodicIconGenerator
	var allStatGenerators []StatGenerator
	var allTabularGenerators []PeriodicTabularGenerator
	var allPlayerStatCalculators []PlayerStatisticCalculator
	var basicHandler basicHandler

	mapGenerator.Setup(header.MapName, imgSize)

	basicHandler.Setup(&p, tickRate, mapMetadata)
	basicHandler.RegisterBasicEvents()
	allTabularGenerators = append(allTabularGenerators, &basicHandler)

	tradeIntervalLimit := 5.0
	var kdatHandler KDATCalculator
	kdatHandler.Register(&basicHandler)
	kdatHandler.Setup(tradeIntervalLimit)
	allPlayerStatCalculators = append(allPlayerStatCalculators, &kdatHandler)

	var adrHandler ADRCalculator
	adrHandler.Register(&basicHandler)
	allPlayerStatCalculators = append(allPlayerStatCalculators, &adrHandler)

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
	infoHandler.Setup(imgSize, updateInterval, rootMatchPath, &allIconGenerators, &allTabularGenerators, &allStatGenerators, &allPlayerStatCalculators)

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

func currentPlayerMappings(gs dem.GameState) map[int]playerMapping {
	newAllPlayers := make(map[int]playerMapping)
	players := gs.Participants().Playing()
	ctCount := 0
	tCount := 0
	playerBasePos := 0
	for _, player := range players {
		isCT := (player.Team == 3)
		isTR := (player.Team == 2)

		if isTR && tCount > 4 || isCT && ctCount > 4 {
			fmt.Println("invalid team size")
			return make(map[int]playerMapping)
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
		newAllPlayers[player.UserID] = playerMapping{playerObject: player, currentSlot: playerBasePos}
	}

	return newAllPlayers
}

func sortPlayersByUserID(allPlayers map[int]playerMapping) []int {

	var keys []int
	for userID := range allPlayers {
		keys = append(keys, userID)
	}
	sort.Ints(keys)
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
