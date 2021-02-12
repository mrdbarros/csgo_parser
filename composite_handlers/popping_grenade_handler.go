package composite_handlers

import (
	"github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
	map_builder "github.com/mrdbarros/csgo_analyze/map_builder"
)

type PoppingGrenadeHandler struct {
	basicHandler   *BasicHandler
	activeGrenades []*grenadeTracker
	baseIcons      map[common.EquipmentType]map_builder.Icon
}

func (ph *PoppingGrenadeHandler) Update() {

}

func (ph *PoppingGrenadeHandler) Register(bh *BasicHandler) error {
	ph.basicHandler = bh
	bh.RegisterGrenadeEventIfSubscriber(interface{}(ph).(GrenadeEventIfSubscriber))
	bh.RegisterRoundStartSubscriber(interface{}(ph).(RoundStartSubscriber))
	return nil
}

func (ph *PoppingGrenadeHandler) RoundStartHandler(e events.RoundStart) {
	ph.activeGrenades = nil
}

//e holds smoke start/expired or inferno start/expired and other grenade events
func (ph *PoppingGrenadeHandler) GrenadeEventIfHandler(e events.GrenadeEventIf) {

	// if molly, incgrenade or smoke
	if e.Base().GrenadeType == common.EqSmoke || e.Base().GrenadeType == common.EqIncendiary || e.Base().GrenadeType == common.EqMolotov {
		eventTime := ph.basicHandler.currentTime
		grenadeEntityID := e.Base().GrenadeEntityID
		if ph.IsTracked(grenadeEntityID) {
			ph.RemoveGrenade(grenadeEntityID)
		} else {
			newGrenade := grenadeTracker{grenadeEvent: e.Base(), grenadeTime: eventTime}
			ph.activeGrenades = append(ph.activeGrenades, &newGrenade)
		}

	}

}

func (ph *PoppingGrenadeHandler) IsTracked(entityID int) bool {
	for _, activeGrenade := range ph.activeGrenades {
		if activeGrenade.grenadeEvent.GrenadeEntityID == entityID {
			return true
		}

	}
	return false
}

func (ph *PoppingGrenadeHandler) GetPeriodicIcons() ([]map_builder.Icon, error) {
	var iconList []map_builder.Icon
	for _, activeGrenade := range ph.activeGrenades {
		newIcon := ph.baseIcons[activeGrenade.grenadeEvent.GrenadeType]
		x, y := activeGrenade.grenadeEvent.Position.X, activeGrenade.grenadeEvent.Position.Y
		newIcon.X, newIcon.Y = x, y
		iconList = append(iconList, newIcon)
	}
	return iconList, nil
}

func (ph *PoppingGrenadeHandler) RemoveGrenade(entityID int) {

	for i, grenade := range ph.activeGrenades {
		if grenade.grenadeEvent.GrenadeEntityID == entityID {
			ph.activeGrenades[i] = ph.activeGrenades[len(ph.activeGrenades)-1]
			ph.activeGrenades = ph.activeGrenades[:(len(ph.activeGrenades) - 1)]
			break
		}
	}
}

func (ph *PoppingGrenadeHandler) SetBaseIcons() {
	ph.baseIcons = map[common.EquipmentType]map_builder.Icon{
		505: map_builder.Icon{IconName: map_builder.SmokeIconName},
		502: map_builder.Icon{IconName: map_builder.IncendiaryIconName},
		503: map_builder.Icon{IconName: map_builder.IncendiaryIconName},
	}
}

type grenadeTracker struct {
	grenadeEvent events.GrenadeEvent
	grenadeTime  float64
}
