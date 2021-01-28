package composite_handlers

import (
	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
	map_builder "github.com/mrdbarros/csgo_analyze/map_builder"
)

type BombHandler struct {
	basicHandler    *BasicHandler
	bombPlanted     bool
	bombPlantedTime float64
	baseIcons       map[string]map_builder.Icon
}

func (bmbh *BombHandler) Register(bh *BasicHandler) error {
	bmbh.basicHandler = bh
	bh.RegisterBombPlantedSubscriber(interface{}(bmbh).(BombPlantedSubscriber))
	bh.RegisterRoundStartSubscriber(interface{}(bmbh).(RoundStartSubscriber))
	return nil
}

func (bmbh *BombHandler) Update() {

}

func (bh *BombHandler) BombPlantedHandler(e events.BombPlanted) {
	bh.bombPlanted = true
	bh.bombPlantedTime = bh.basicHandler.currentTime

}

func (bh *BombHandler) RoundStartHandler(e events.RoundStart) {

	bh.bombPlanted = false

}

func (bh *BombHandler) GetPeriodicIcons() (icons []map_builder.Icon, err error) {
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
	icons = append(icons, map_builder.Icon{IconName: icon, X: x, Y: y})
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
