package composite_handlers

import (
	"github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
	map_builder "github.com/mrdbarros/csgo_analyze/map_builder"
)

type BombHandler struct {
	statisticHolder
	bombPlanted     bool
	bombPlantedTime float64
	bombDefused     bool
	baseIcons       map[string]map_builder.Icon
	bombCarrier     *common.Player
}

func (bmbh *BombHandler) Register(bh *BasicHandler) error {
	bmbh.basicHandler = bh
	bh.RegisterBombPlantedSubscriber(interface{}(bmbh).(BombPlantedSubscriber))
	bh.RegisterRoundStartSubscriber(interface{}(bmbh).(RoundStartSubscriber))
	bh.RegisterBombDefusedSubscriber(interface{}(bmbh).(BombDefusedSubscriber))
	bh.RegisterBombDroppedSubscriber(interface{}(bmbh).(BombDroppedSubscriber))
	bh.RegisterBombPickupSubscriber(interface{}(bmbh).(BombPickupSubscriber))
	bh.RegisterRoundFreezetimeEndSubscriber(interface{}(bmbh).(RoundFreezetimeEndSubscriber))
	bmbh.baseStatsHeaders = []string{"Bombs Planted", "Bombs Picked Up", "Bombs Defused", "Bombs Dropped"}
	bmbh.defaultValues = make(map[string]float64)
	return nil
}

func (bmbh *BombHandler) Update() {

}

func (bh *BombHandler) BombPlantedHandler(e events.BombPlanted) {
	bh.bombPlanted = true
	bh.bombPlantedTime = bh.basicHandler.currentTime
	bh.addToPlayerStat(e.Player, 1, "Bombs Planted")
}

func (bh *BombHandler) RoundStartHandler(e events.RoundStart) {

	bh.bombPlanted = false
	bh.bombDefused = false

}

func (bh *BombHandler) BombDefusedHandler(e events.BombDefused) {

	bh.bombDefused = true
	bh.addToPlayerStat(e.Player, 1, "Bombs Defused")
}

func (bh *BombHandler) BombDroppedHandler(e events.BombDropped) {

	bh.addToPlayerStat(e.Player, 1, "Bombs Dropped")
	bh.bombCarrier = nil
}

func (bh *BombHandler) BombPickupHandler(e events.BombPickup) {
	bh.bombCarrier = e.Player
	if bh.bombCarrier != nil {
		bh.addToPlayerStat(bh.bombCarrier, 1, "Bombs Dropped")
	}
	bh.addToPlayerStat(e.Player, 1, "Bombs Picked Up")
}

func (bh *BombHandler) RoundFreezetimeEndHandler(e events.RoundFreezetimeEnd) {
	bh.bombCarrier = nil
	if bh.basicHandler.roundNumber-1 < len(bh.playerStats) {
		bh.playerStats = bh.playerStats[:bh.basicHandler.roundNumber-1]
	}
	bh.AddNewRound()
}

func (bh *BombHandler) GetPeriodicIcons() (icons []map_builder.Icon, err error) {
	parser := (*bh.basicHandler.parser)
	bomb := parser.GameState().Bomb()
	var icon string
	if bh.bombDefused {
		icon = "bomb_defused"
	} else if bh.bombPlanted {
		icon = "bomb_planted"
	} else if bomb.Carrier == nil {
		icon = "bomb_dropped"
	} else {
		icon = "c4_carrier"
	}
	bombPosition := bomb.Position()
	icons = append(icons, map_builder.Icon{IconName: icon, X: bombPosition.X, Y: bombPosition.Y})
	return icons, nil
}

func (bh *BombHandler) GetPeriodicTabularData() ([]string, []float64, error) {
	newCSVRow := []float64{0}
	if bh.bombPlanted {
		newCSVRow[0] = bh.basicHandler.currentTime - bh.bombPlantedTime
	}

	header := []string{"bomb_timeticking"}
	return header, newCSVRow, nil
}
