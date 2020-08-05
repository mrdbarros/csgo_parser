package utils

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
	tIcon := Icon{X: 200, Y: 300, IconName: "terrorist_1"}
	ctIcon := Icon{X: 100, Y: 400, IconName: "ct_1"}
	annMap := AnnotatedMap{IconsList: nil, SourceMap: "de_train"}
	annMap.IconsList = append(annMap.IconsList, tIcon)
	annMap.IconsList = append(annMap.IconsList, ctIcon)
	img := DrawMap(annMap)
	jpeg.Encode(third, img, &jpeg.Options{jpeg.DefaultQuality})
	defer third.Close()

}

func main() {
	PadLeft("2", "0", 2)
}
