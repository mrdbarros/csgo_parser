package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"

	dem "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs"
	metadata "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/metadata"
	"github.com/mrdbarros/csgo_analyze/composite_handlers"
	map_builder "github.com/mrdbarros/csgo_analyze/map_builder"
	utils "github.com/mrdbarros/csgo_analyze/utils"
)

//var allPlayers = make(map[int]*playerMapping)

func ProcessDemoFile(demPath string, fileID int, destDir string, tickRate int) {
	fileStat, err := os.Stat(demPath)

	f, err := os.Open(demPath)
	utils.CheckError(err)
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Erro no processamento do arquivo!", r)
		}
	}()
	fileName := filepath.Base(demPath)
	fmt.Println("Processing demo: ", fileName)
	hasher := sha256.New()

	_, err = io.Copy(hasher, f)
	utils.CheckError(err)
	defer f.Close()

	f, err = os.Open(demPath)
	utils.CheckError(err)

	p := dem.NewParser(f)

	header, err := p.ParseHeader()
	utils.CheckError(err)

	fmt.Println("Map:", header.MapName)
	rootMatchPath := destDir + "/" + header.MapName + "/" + strconv.Itoa(fileID)
	dirExists, _ := utils.Exists(rootMatchPath)
	if !dirExists {
		err = os.MkdirAll(rootMatchPath, 0700)
		utils.CheckError(err)
	}

	imgSize := 300

	mapMetadata := metadata.MapNameToMap[header.MapName]
	var mapGenerator map_builder.MapGenerator
	var allIconGenerators []composite_handlers.PeriodicIconGenerator
	var allStatGenerators []composite_handlers.StatGenerator
	var allTabularGenerators []composite_handlers.PeriodicTabularGenerator
	var allPlayerStatCalculators []composite_handlers.PlayerStatisticCalculator
	var basicHandler composite_handlers.BasicHandler

	mapGenerator.Setup(header.MapName, imgSize)

	basicHandler.Setup(&p, tickRate, mapMetadata, fileStat.ModTime(), fileName)
	basicHandler.RegisterBasicEvents()
	allTabularGenerators = append(allTabularGenerators, &basicHandler)
	allPlayerStatCalculators = append(allPlayerStatCalculators, &basicHandler)

	tradeIntervalLimit := 3.0
	var kdatHandler composite_handlers.KDATCalculator
	kdatHandler.Register(&basicHandler)
	kdatHandler.Setup(tradeIntervalLimit)
	allPlayerStatCalculators = append(allPlayerStatCalculators, &kdatHandler)

	var adrHandler composite_handlers.ADRCalculator
	adrHandler.Register(&basicHandler)
	allPlayerStatCalculators = append(allPlayerStatCalculators, &adrHandler)

	var flashCalc composite_handlers.FlashUsageCalculator
	flashCalc.Register(&basicHandler)
	allPlayerStatCalculators = append(allPlayerStatCalculators, &flashCalc)

	var popHandler composite_handlers.PoppingGrenadeHandler
	popHandler.SetBaseIcons()
	popHandler.Register(&basicHandler)
	allIconGenerators = append(allIconGenerators, &popHandler)

	var bmbHandler composite_handlers.BombHandler
	bmbHandler.Register(&basicHandler)
	allTabularGenerators = append(allTabularGenerators, &bmbHandler)
	allIconGenerators = append(allIconGenerators, &bmbHandler)

	var playerHandler composite_handlers.PlayerPeriodicInfoHandler
	playerHandler.Register(&basicHandler)
	allTabularGenerators = append(allTabularGenerators, &playerHandler)
	allIconGenerators = append(allIconGenerators, &playerHandler)

	var infoHandler composite_handlers.InfoGenerationHandler
	updateInterval := 1.5 //1.5s between framegroups
	infoHandler.Register(&basicHandler)
	infoHandler.Setup(imgSize, updateInterval, rootMatchPath, hex.EncodeToString(hasher.Sum(nil)),
		&allIconGenerators, &allTabularGenerators, &allStatGenerators, &allPlayerStatCalculators)

	err = p.ParseToEnd()
	p.Close()

	utils.CheckError(err)

	// Parse to end
}

func main() {
	demPath := os.Args[2]
	destDir := os.Args[3]

	mode := flag.String("mode", "file", "process mode (file/dir)")
	fileID := 0
	tickRate, _ := strconv.Atoi(os.Args[4])
	flag.Parse()
	if *mode == "file" {
		ProcessDemoFile(demPath, fileID, destDir, tickRate)
	} else if *mode == "dir" {

		filepath.Walk(demPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
				return err
			}
			if info.IsDir() {
				return nil
			}
			ProcessDemoFile(path, fileID, destDir, tickRate)
			fileID++
			return nil
		})

	} else {
		log.Fatal("invalid mode.")
	}

}
