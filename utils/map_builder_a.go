package main

import (
	"image/jpeg"
	"log"
	"os"
)

func DrawMapTest() {
	third, err := os.Create("/home/marcel/projetos/data/csgo_analyze/images/output_map.jpg")
	if err != nil {
		log.Fatalf("failed to create: %s", err)
	}
	tIcon := Icon{x: 200, y: 300, iconName: "terrorist_1"}
	ctIcon := Icon{x: 100, y: 400, iconName: "ct_1"}
	annMap := AnnotatedMap{iconsList: nil, sourceMap: "de_train"}
	annMap.iconsList = append(annMap.iconsList, tIcon)
	annMap.iconsList = append(annMap.iconsList, ctIcon)
	img := DrawMap(annMap)
	jpeg.Encode(third, img, &jpeg.Options{jpeg.DefaultQuality})
	defer third.Close()

}

func main() {
	DrawMapTest()
}
