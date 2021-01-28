package composite_handlers

import (
	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
	map_builder "github.com/mrdbarros/csgo_analyze/map_builder"
)

//generic event handler registering interface
type CompositeEventHandler interface {
	Register(*BasicHandler) error
	//Unregister() error
}

type PeriodicGenerator interface {
	CompositeEventHandler
	Update()
}

//IconGenerators generate icons on output map
type PeriodicIconGenerator interface {
	PeriodicGenerator
	GetPeriodicIcons() ([]map_builder.Icon, error)
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

//Interface to PlayerDisconnected event subscribers
type PlayerDisconnectedSubscriber interface {
	PlayerDisconnectedHandler(events.PlayerDisconnected)
}
