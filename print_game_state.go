package main

import (
	"fmt"
	"os"
	"strconv"
	dem "github.com/markus-wa/demoinfocs-golang"
	//ex "github.com/markus-wa/demoinfocs-golang/examples"
	events "github.com/markus-wa/demoinfocs-golang/events"
)
// exists returns whether the given file or directory exists
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil { return true, nil }
	if os.IsNotExist(err) { return false, nil }
        return true, err
}
func processDemoFile(demPath string,file_id int){
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
	dir_name:="/home/jupyter/mrdbarros/csgo_analyze/data/"+header.MapName
	dir_exists,_:=exists(dir_name)
	if !dir_exists{
		err=os.Mkdir(dir_name,0700)
		checkError(err)
	}
	new_file:=dir_name+"/"+header.MapName+"_"+ strconv.Itoa(file_id)+".txt"
	file_write, err := os.Create(new_file)
	checkError(err)
	current_state:=""

	defer file_write.Close()
	
	p.RegisterEventHandler(func(e events.Kill) {
		killer_team := e.Killer.Team
		if killer_team == 2 {
			current_state+="t_kill "
		} else if killer_team == 3 {
			current_state+="ct_kill "
		}
	})
	
	p.RegisterEventHandler(func(e events.RoundEnd) {
		
		win_team := e.Winner
		if win_team == 2 {
			current_state+="t_round_win "
		} else if win_team == 3 {
			current_state+="ct_round_win "
		} else {
			current_state+="invalid_round_end "
		}
	})
	
	p.RegisterEventHandler(func(e events.RoundStart) {
		current_state+="round_start "
	})
	
	p.RegisterEventHandler(func(e events.RoundEndOfficial) {

		current_state+="round_end_official "
	})
	p.RegisterEventHandler(func(e events.BombPlanted) {

		current_state+="bomb_planted "
	})
	p.RegisterEventHandler(func(e events.BombPlantBegin) {

		current_state+="bomb_plant_begin "
	})
	err = p.ParseToEnd()
	checkError(err)

	_, err = file_write.WriteString(current_state)
	checkError(err)
	// Parse to end
}

// Run like this: go run print_events.go -demo /path/to/demo.dem
func main() {
	dem_path:= os.Args[1]
	file_id_str:=os.Args[2]
	file_id, err := strconv.Atoi(file_id_str)
	checkError(err)
	processDemoFile(dem_path,file_id)
}


func checkError(err error) {
	if err != nil {
		print("error!")
		panic(err)
	}
}
