package main

import (
	"fmt"
	"os"
	"strconv"

	r3 "github.com/golang/geo/r3"
	dem "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
)

var currentState = ""
var gameReset = false
var gameStarted = false
var discretizeFactor = 20.0
var roundStartTime int
var lastUpdate = 0
var lastTimeEvent = 0
var tickRate = 0
var posUpdateInterval = 2

type playerMapping struct {
	playerSeqID int
	position    r3.Vector
}

var trMap = make(map[int]playerMapping)
var ctMap = make(map[int]playerMapping)

// exists returns whether the given file or directory exists
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func checkError(err error) {
	if err != nil {
		print("error!")
		panic(err)
	}
}

func processDemoFile(demPath string, fileID int, destDir string) {
	f, err := os.Open(demPath)
	checkError(err)
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Erro no processamento do arquivo!", r)
		}
	}()
	p := dem.NewParser(f)
	defer f.Close()
	header, err := p.ParseHeader()
	checkError(err)
	fmt.Println("Map:", header.MapName)
	mapName := header.MapName
	dirName := destDir + "/" + header.MapName
	dirExists, _ := exists(dirName)
	if !dirExists {
		err = os.Mkdir(dirName, 0700)
		checkError(err)
	}
	newFile := dirName + "/" + header.MapName + "_" + strconv.Itoa(fileID) + ".txt"
	fileWrite, err := os.Create(newFile)
	checkError(err)

	defer fileWrite.Close()

	p.RegisterEventHandler(func(e events.FrameDone) {
		gs := p.GameState()
		if !(gs == nil) {
			processFrameEnd(gs, p)
		}

	})

	err = p.ParseToEnd()
	checkError(err)
	if currentState[0:3] != "de_" {
		currentState = mapName + " " + currentState
	}
	_, err = fileWrite.WriteString(currentState)
	checkError(err)
	// Parse to end
}

func main() {
	demPath := os.Args[1]
	fileIDStr := os.Args[2]
	destDir := os.Args[3]
	fileID, err := strconv.Atoi(fileIDStr)
	checkError(err)
	tickRate, err = strconv.Atoi(os.Args[4])
	checkError(err)
	processDemoFile(demPath, fileID, destDir)
}

func processFrameEnd(gs dem.GameState, p dem.Parser) {
	//print(p.Header().PlaybackFrames)
	if getRoundTime(p)%posUpdateInterval == 0 && getCurrentTime(p) != lastUpdate {
		lastUpdate = getCurrentTime(p)
		processPlayerPositions(p)
	}
}

func getRoundTime(p dem.Parser) int {
	return int(getCurrentTime(p) - roundStartTime)
}

func getCurrentTime(p dem.Parser) int {
	return p.CurrentFrame() / tickRate
}

func processPlayerPositions(p dem.Parser) {
	gs := p.GameState()
	tr := gs.TeamTerrorists()
	ct := gs.TeamCounterTerrorists()
	fmt.Println(ct)
	fmt.Println(tr)
}
