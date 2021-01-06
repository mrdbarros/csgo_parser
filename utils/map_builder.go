package utils

import (
	"image"
	"image/color"
	"image/draw"
	_ "image/png" //png thru image.decode
	"os"

	"github.com/disintegration/imaging"
)

//const for names because go doesn't have proper enum
const SmokeIconName = "smoke"
const IncendiaryIconName = "incendiary"

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
		"smoke":        "/home/marcel/projetos/data/csgo_analyze/images/icons/smoke.png",
		"flash":        "flash",
		"incendiary":   "/home/marcel/projetos/data/csgo_analyze/images/icons/incendiary.png",
		"terrorist_1":  "/home/marcel/projetos/data/csgo_analyze/images/icons/t.png",
		"terrorist_2":  "/home/marcel/projetos/data/csgo_analyze/images/icons/t.png",
		"terrorist_3":  "/home/marcel/projetos/data/csgo_analyze/images/icons/t.png",
		"terrorist_4":  "/home/marcel/projetos/data/csgo_analyze/images/icons/t.png",
		"terrorist_5":  "/home/marcel/projetos/data/csgo_analyze/images/icons/t.png",
		"ct_1":         "/home/marcel/projetos/data/csgo_analyze/images/icons/ct.png",
		"ct_2":         "/home/marcel/projetos/data/csgo_analyze/images/icons/ct.png",
		"ct_3":         "/home/marcel/projetos/data/csgo_analyze/images/icons/ct.png",
		"ct_4":         "/home/marcel/projetos/data/csgo_analyze/images/icons/ct.png",
		"ct_5":         "/home/marcel/projetos/data/csgo_analyze/images/icons/ct.png",
		"he":           "he",
		"bomb_planted": "/home/marcel/projetos/data/csgo_analyze/images/icons/planted_c4.png",
		"bomb_dropped": "/home/marcel/projetos/data/csgo_analyze/images/icons/dropped_c4.png",
		"kit":          "/home/marcel/projetos/data/csgo_analyze/images/icons/kit_carrier.png",
		"c4_carrier":   "/home/marcel/projetos/data/csgo_analyze/images/icons/c4_carrier.png",
		"1":            "/home/marcel/projetos/data/csgo_analyze/images/icons/1.png",
		"2":            "/home/marcel/projetos/data/csgo_analyze/images/icons/2.png",
		"3":            "/home/marcel/projetos/data/csgo_analyze/images/icons/3.png",
		"4":            "/home/marcel/projetos/data/csgo_analyze/images/icons/4.png",
		"5":            "/home/marcel/projetos/data/csgo_analyze/images/icons/5.png",
	}
}

//Icon represents a single icon and its position
type Icon struct {
	X        float64
	Y        float64
	IconName string
	Rotate   float64
}

type MapGenerator struct {
	mapImage   *image.NRGBA
	iconGetter func(Icon) *(image.Image)
	imgSize    int
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

func (mapGenerator *MapGenerator) Setup(mapName string, imgSize int) {

	iconNameToPath := getIconNameToImageMap()
	mapGenerator.iconGetter = iconImageGetter(iconNameToPath)
	mapGenerator.imgSize = imgSize
	mapPath := getMapsToImageMap()[mapName]

	// Load map overview image
	fMap, err := os.Open(mapPath)
	checkError(err)
	imgMap, _, err := image.Decode(fMap)
	checkError(err)

	// Create output canvas and use map overview image as base
	img := image.NewNRGBA(imgMap.Bounds())
	draw.Draw(img, imgMap.Bounds(), imgMap, image.ZP, draw.Over)
	mapGenerator.mapImage = img
}

//DrawMap uses iconLists and mapGenerator to generate all maps from a round
func (mapGenerator MapGenerator) DrawMap(iconLists [][]Icon) []*(image.NRGBA) {
	var imgLocation image.Rectangle
	var baseImage *image.NRGBA
	var roundImages []*image.NRGBA
	for _, iconList := range iconLists {
		if len(iconList) != 0 {
			baseImage = image.NewNRGBA(mapGenerator.mapImage.Bounds())
			draw.Draw(baseImage, mapGenerator.mapImage.Bounds(), mapGenerator.mapImage, image.ZP, draw.Over)
			for _, icon := range iconList {
				iconImg := *mapGenerator.iconGetter(icon)

				if icon.Rotate != 0.0 {
					iconImg = imaging.Rotate(iconImg, icon.Rotate, color.Transparent)

				}
				offset := image.Pt(int(icon.X)-iconImg.Bounds().Max.X/2, int(icon.Y)-iconImg.Bounds().Max.Y/2)
				imgLocation = iconImg.Bounds().Add(offset)
				draw.Draw(baseImage, imgLocation, iconImg, image.ZP, draw.Over)
			}
			roundImages = append(roundImages, baseImage)

		}

	}

	return roundImages
}
