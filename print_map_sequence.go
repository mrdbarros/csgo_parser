package main

import (
	"fmt"
	"os"
	"strconv"

	r3 "github.com/golang/geo/r3"
	dem "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
)

var current_state = ""
var game_reset = false
var game_started = false
var discretize_factor = 20.0
var round_start_time int
var last_update = 0
var last_time_event = 0
var tick_rate = 0
var pos_update_interval = 2

type player_mapping struct {
	player_seq_id int
	position      r3.Vector
}

var tr_map = make(map[int]player_mapping)
var ct_map = make(map[int]player_mapping)

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

func processDemoFile(demPath string, file_id int, dest_dir string) {
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
	map_name := header.MapName
	dir_name := dest_dir + "/" + header.MapName
	dir_exists, _ := exists(dir_name)
	if !dir_exists {
		err = os.Mkdir(dir_name, 0700)
		checkError(err)
	}
	new_file := dir_name + "/" + header.MapName + "_" + strconv.Itoa(file_id) + ".txt"
	file_write, err := os.Create(new_file)
	checkError(err)

	defer file_write.Close()

	p.RegisterEventHandler(func(e events.FrameDone) {
		gs := p.GameState()
		if !(gs == nil) {
			processFrameEnd(gs, p)
		}

	})

	err = p.ParseToEnd()
	checkError(err)
	if current_state[0:3] != "de_" {
		current_state = map_name + " " + current_state
	}
	_, err = file_write.WriteString(current_state)
	checkError(err)
	// Parse to end
}

func main() {
	dem_path := os.Args[1]
	file_id_str := os.Args[2]
	dest_dir := os.Args[3]
	file_id, err := strconv.Atoi(file_id_str)
	checkError(err)
	tick_rate, err = strconv.Atoi(os.Args[4])
	checkError(err)
	processDemoFile(dem_path, file_id, dest_dir)
}

func processFrameEnd(gs dem.GameState, p dem.Parser) {
	//print(p.Header().PlaybackFrames)
	if getRoundTime(p)%pos_update_interval == 0 && getCurrentTime(p) != last_update {
		last_update = getCurrentTime(p)
		processPlayerPositions(p)
	}
}

func getRoundTime(p dem.Parser) int {
	return int(getCurrentTime(p) - round_start_time)
}

func getCurrentTime(p dem.Parser) int {
	return p.CurrentFrame() / tick_rate
}

func processPlayerPositions(p dem.Parser) {
	gs := p.GameState()
	tr := gs.TeamTerrorists()
	ct := gs.TeamCounterTerrorists()
	fmt.Println(ct)
	fmt.Println(tr)
}
