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
	"sync"

	"github.com/mrdbarros/csgo_analyze/database"

	dem "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs"
	metadata "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/metadata"
	"github.com/mrdbarros/csgo_analyze/composite_handlers"

	utils "github.com/mrdbarros/csgo_analyze/utils"
)

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

	hashString := hex.EncodeToString(hasher.Sum(nil))

	dbConn := database.OpenDBConn()
	skipProcessedFile := false

	f, err = os.Open(demPath)
	utils.CheckError(err)

	p := dem.NewParser(f)

	header, err := p.ParseHeader()
	utils.CheckError(err)

	fmt.Println("Map:", header.MapName)
	rootMatchPath := destDir + "/" + header.MapName + "/" + hashString
	dirExists, _ := utils.Exists(rootMatchPath)

	if database.CheckIfProcessed(dbConn, hashString) && skipProcessedFile && dirExists {
		fmt.Println("Demo already processed, skipping...")
		dbConn.Close()
		return
	} else {
		dbConn.Close()
	}

	if !dirExists {
		err = os.MkdirAll(rootMatchPath, 0700)
		utils.CheckError(err)
	}

	imgSize := 800

	mapMetadata := metadata.MapNameToMap[header.MapName]
	var allIconGenerators []composite_handlers.PeriodicIconGenerator
	var allStatGenerators []composite_handlers.StatGenerator
	var allTabularGenerators []composite_handlers.PeriodicTabularGenerator
	var allPlayerStatCalculators []composite_handlers.PlayerStatisticCalculator
	var basicHandler composite_handlers.BasicHandler

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

	var bmbHandler composite_handlers.BombHandler
	bmbHandler.Register(&basicHandler)
	allTabularGenerators = append(allTabularGenerators, &bmbHandler)
	allPlayerStatCalculators = append(allPlayerStatCalculators, &bmbHandler)

	var playerHandler composite_handlers.PlayerPeriodicInfoHandler
	playerHandler.Register(&basicHandler)
	allTabularGenerators = append(allTabularGenerators, &playerHandler)

	generateIcons := false
	if generateIcons {
		allIconGenerators = append(allIconGenerators, &popHandler)
		allIconGenerators = append(allIconGenerators, &bmbHandler)
		allIconGenerators = append(allIconGenerators, &playerHandler)
		allIconGenerators = append(allIconGenerators, &flashCalc)
	}

	var infoHandler composite_handlers.InfoGenerationHandler
	updateInterval := 2.0 //# of seconds between framegroups
	infoHandler.Register(&basicHandler)
	infoHandler.Setup(imgSize, updateInterval, rootMatchPath, hashString,
		&allIconGenerators, &allTabularGenerators, &allStatGenerators, &allPlayerStatCalculators)

	err = p.ParseToEnd()
	p.Close()

	utils.CheckError(err)

	// Parse to end
}

type demoFile struct {
	demPath  string
	fileID   int
	destDir  string
	tickRate int
}

func worker(wg *sync.WaitGroup, jobChan <-chan demoFile) {
	defer wg.Done()
	for demFile := range jobChan {
		fmt.Println("Demos left:", len(jobChan))
		ProcessDemoFile(demFile.demPath, demFile.fileID, demFile.destDir, demFile.tickRate)
	}
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
		workerCount := 7
		// use a WaitGroup
		var wg sync.WaitGroup

		// make a channel with a capacity of 100.
		jobChan := make(chan demoFile, 500)

		for i := 0; i < workerCount; i++ {
			go worker(&wg, jobChan)
		}
		wg.Add(workerCount)

		filepath.Walk(demPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
				return err
			}
			if info.IsDir() {
				return nil
			}
			// enqueue a job
			jobChan <- demoFile{demPath: path, fileID: fileID, destDir: destDir, tickRate: tickRate}
			fileID++
			return nil
		})
		wg.Wait()

	} else {
		log.Fatal("invalid mode.")
	}

}
