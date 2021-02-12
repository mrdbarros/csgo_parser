package composite_handlers

import (
	"strconv"

	"github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	metadata "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/metadata"
	map_builder "github.com/mrdbarros/csgo_analyze/map_builder"
	utils "github.com/mrdbarros/csgo_analyze/utils"
)

type PlayerPeriodicInfoHandler struct {
	basicHandler                *BasicHandler
	periodicTabularInfoGatherer []IPlayersPeriodicTabularInfoGatherer
	periodicPlayerIconGatherer  []IPeriodicPlayerIconGatherer
}

func (ph *PlayerPeriodicInfoHandler) Register(bh *BasicHandler) error {
	ph.basicHandler = bh

	bg := new(basicPlayerPositionGatherer)
	bg.Setup(bh)
	ph.periodicPlayerIconGatherer = append(ph.periodicPlayerIconGatherer, bg)
	ph.periodicTabularInfoGatherer = append(ph.periodicTabularInfoGatherer, new(hpGatherer))
	ph.periodicTabularInfoGatherer = append(ph.periodicTabularInfoGatherer, new(currentFlashTimeGatherer))
	ph.periodicTabularInfoGatherer = append(ph.periodicTabularInfoGatherer, new(weaponsGatherer))
	return nil
}

func (ph *PlayerPeriodicInfoHandler) Update() {
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

func (ph *PlayerPeriodicInfoHandler) updatePlayerInfo(playerInfoGatherers []IPeriodicPlayerInfoGatherer) {

	for _, playerMapping := range ph.basicHandler.playerMappings[ph.basicHandler.roundNumber-1] {
		player := playerMapping.playerObject

		for _, playerGatherer := range playerInfoGatherers {
			playerGatherer.updatePlayer(player, playerMapping.currentSlot)
		}

	}

}

func (ph *PlayerPeriodicInfoHandler) GetPeriodicTabularData() (newHeader []string, newCSVRow []float64, err error) {

	for _, periodicTabularGatherer := range ph.periodicTabularInfoGatherer {

		tempHeader, tempCSV := periodicTabularGatherer.GetPeriodicTabularInfo()
		newCSVRow = append(newCSVRow, tempCSV...)
		newHeader = append(newHeader, tempHeader...)
	}

	return newHeader, newCSVRow, err
}

func (ph *PlayerPeriodicInfoHandler) GetPeriodicIcons() ([]map_builder.Icon, error) {
	var iconList []map_builder.Icon
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
	GetPlayerIcons() []map_builder.Icon
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

type playersIconGatherer struct {
	playersIcons []map_builder.Icon
	mapMetadata  metadata.Map
}

type basicPlayerPositionGatherer struct {
	playersIconGatherer playersIconGatherer
}

func (bg *basicPlayerPositionGatherer) Init() {
	bg.playersIconGatherer.playersIcons = nil
}

func (bg *basicPlayerPositionGatherer) Setup(basicHandler *BasicHandler) {
	bg.playersIconGatherer.mapMetadata = basicHandler.mapMetadata
}

func (bg *basicPlayerPositionGatherer) updatePlayer(player *common.Player, basePos int) {

	if player.Health() > 0 {
		x, y := player.Position().X, player.Position().Y
		var icon string

		if basePos/5 == 1 {

			icon = "ct_"
			if player.HasDefuseKit() {
				newIcon := map_builder.Icon{X: x, Y: y, IconName: "kit"} //t or ct icon
				bg.playersIconGatherer.playersIcons = append(bg.playersIconGatherer.playersIcons, newIcon)
			}

		} else {
			icon = "terrorist_"

		}
		playerNumber := basePos%5 + 1 //count 1-5 tr, 6-10 ct

		newIcon := map_builder.Icon{X: x, Y: y, IconName: icon + strconv.Itoa(playerNumber), Rotate: float64(player.ViewDirectionX())} //t or ct icon
		bg.playersIconGatherer.playersIcons = append(bg.playersIconGatherer.playersIcons, newIcon)
		newIcon = map_builder.Icon{X: x, Y: y, IconName: strconv.Itoa(playerNumber)}
		bg.playersIconGatherer.playersIcons = append(bg.playersIconGatherer.playersIcons, newIcon)
	}

}

func (bg *basicPlayerPositionGatherer) GetPlayerIcons() []map_builder.Icon {
	return bg.playersIconGatherer.playersIcons
}
