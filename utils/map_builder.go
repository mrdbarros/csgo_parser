package main

import (
	"image"
	"image/draw"
	"os"
)

//returns map from mapName to mapImagePath
func getMapsToImageMap() map[string]string {
	return map[string]string{
		"de_dust2":    "de_dust2",
		"de_inferno":  "de_inferno",
		"de_nuke":     "de_nuke",
		"de_mirage":   "de_mirage",
		"de_vertigo":  "de_vertigo",
		"de_overpass": "de_overpass",
		"de_cache":    "de_cache",
		"de_train":    "/home/marcel/projetos/data/csgo_analyze/images/maps/de_train.jpg",
	}
}

//returns map from iconName to iconImagePath
func getIconNameToImageMap() map[string]string {
	return map[string]string{
		"smoke":        "smoke",
		"flash":        "flash",
		"molotov":      "molotov",
		"terrorist_1":  "/home/marcel/projetos/data/csgo_analyze/images/icons/t.jpg",
		"terrorist_2":  "terrorist_2",
		"terrorist_3":  "terrorist_3",
		"terrorist_4":  "terrorist_4",
		"terrorist_5":  "terrorist_5",
		"ct_1":         "/home/marcel/projetos/data/csgo_analyze/images/icons/ct.jpg",
		"ct_2":         "ct_2",
		"ct_3":         "ct_3",
		"ct_4":         "ct_4",
		"ct_5":         "ct_5",
		"he":           "he",
		"bomb_planted": "bomb_planted",
		"bomb_droped":  "bomb_droped",
	}
}

//Single icon
type Icon struct {
	x        int
	y        int
	iconName string
}

//map with iconlist
type AnnotatedMap struct {
	iconsList []Icon
	sourceMap string
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func iconImageGetter(iconNameToPathMap map[string]string) func(Icon) *(image.Image) {
	loadedImages := make(map[string]*(image.Image))
	iconNameToPathMapInner := iconNameToPathMap
	return func(icon Icon) *image.Image {
		iconImg, ok := loadedImages[icon.iconName]
		if !ok {
			fIcon, err := os.Open(iconNameToPathMapInner[icon.iconName])
			checkError(err)
			newImg, _, err := image.Decode(fIcon)
			checkError(err)
			loadedImages[icon.iconName] = &newImg
			iconImg = loadedImages[icon.iconName]
		}
		return iconImg
	}
}

//uses annotatedMap struct to generate a full image
func DrawMap(annMap AnnotatedMap) *(image.RGBA) {

	mapPath := getMapsToImageMap()[annMap.sourceMap]

	iconNameToPath := getIconNameToImageMap()
	// Load map overview image
	fMap, err := os.Open(mapPath)
	checkError(err)
	imgMap, _, err := image.Decode(fMap)
	checkError(err)

	// Create output canvas and use map overview image as base
	img := image.NewRGBA(imgMap.Bounds())
	draw.Draw(img, imgMap.Bounds(), imgMap, image.ZP, draw.Over)
	iconGetter := iconImageGetter(iconNameToPath)
	for _, icon := range annMap.iconsList {
		offset := image.Pt(icon.x, icon.y)
		iconImg := *iconGetter(icon)
		draw.Draw(img, iconImg.Bounds().Add(offset), iconImg, image.ZP, draw.Over)
	}

	return img
}
