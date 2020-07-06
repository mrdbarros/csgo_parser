package utils

import (
	"image"
	"image/draw"
	"os"
)

//returns map from mapName to mapImagePath
func getMapsToImageMap() map[string]string {
	return map[string]string{
		"de_dust2":    "/home/marcel/projetos/data/csgo_analyze/images/maps/de_dust2.jpg",
		"de_inferno":  "/home/marcel/projetos/data/csgo_analyze/images/maps/de_inferno.jpg",
		"de_nuke":     "/home/marcel/projetos/data/csgo_analyze/images/maps/de_nuke.jpg",
		"de_mirage":   "/home/marcel/projetos/data/csgo_analyze/images/maps/de_mirage.jpg",
		"de_vertigo":  "/home/marcel/projetos/data/csgo_analyze/images/maps/de_vertigo.jpg",
		"de_overpass": "/home/marcel/projetos/data/csgo_analyze/images/maps/de_overpass.jpg",
		"de_cache":    "/home/marcel/projetos/data/csgo_analyze/images/maps/de_cache.jpg",
		"de_train":    "/home/marcel/projetos/data/csgo_analyze/images/maps/de_train.jpg",
	}
}

//returns map from iconName to iconImagePath
func getIconNameToImageMap() map[string]string {
	return map[string]string{
		"smoke":        "/home/marcel/projetos/data/csgo_analyze/images/icons/smoke.jpg",
		"flash":        "flash",
		"incendiary":   "/home/marcel/projetos/data/csgo_analyze/images/icons/incendiary.jpg",
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
		"bomb_planted": "/home/marcel/projetos/data/csgo_analyze/images/icons/bomb_planted.jpg",
		"bomb_dropped": "bomb_dropped",
	}
}

//Icon represents a single icon and its position
type Icon struct {
	X        float64
	Y        float64
	IconName string
}

//AnnotatedMap is a mapName with iconlist
type AnnotatedMap struct {
	IconsList []Icon
	SourceMap string
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
		iconImg, ok := loadedImages[icon.IconName]
		if !ok {
			fIcon, err := os.Open(iconNameToPathMapInner[icon.IconName])
			checkError(err)
			newImg, _, err := image.Decode(fIcon)
			checkError(err)
			loadedImages[icon.IconName] = &newImg
			iconImg = loadedImages[icon.IconName]
		}
		return iconImg
	}
}

//DrawMap uses annotatedMap struct to generate a full image
func DrawMap(annMap AnnotatedMap) *(image.RGBA) {

	mapPath := getMapsToImageMap()[annMap.SourceMap]

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
	for _, icon := range annMap.IconsList {
		offset := image.Pt(int(icon.X), int(icon.Y))
		iconImg := *iconGetter(icon)
		draw.Draw(img, iconImg.Bounds().Add(offset), iconImg, image.ZP, draw.Over)
	}

	return img
}
