package main

import (
	"fmt"
	"os"
	"strconv"
	dem "github.com/markus-wa/demoinfocs-golang"
	//ex "github.com/markus-wa/demoinfocs-golang/examples"
	events "github.com/markus-wa/demoinfocs-golang/events"
	common "github.com/markus-wa/demoinfocs-golang/common"
	"strings"
)

var current_state=""
var game_reset = false
var game_started = false
type player_pos struct {
	x_pos int
	y_pos int
	z_pos int
}




// exists returns whether the given file or directory exists
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil { return true, nil }
	if os.IsNotExist(err) { return false, nil }
        return true, err
}
func processDemoFile(demPath string,file_id int,dest_dir string){
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
	map_name:=header.MapName
	dir_name:=dest_dir+"/"+header.MapName
	dir_exists,_:=exists(dir_name)
	if !dir_exists{
		err=os.Mkdir(dir_name,0700)
		checkError(err)
	}
	new_file:=dir_name+"/"+header.MapName+"_"+ strconv.Itoa(file_id)+".txt"
	file_write, err := os.Create(new_file)
	checkError(err)
	

	defer file_write.Close()
	
	
	p.RegisterEventHandler(func(e events.Kill) {
		if !(e.Killer == nil) {
			killer_team := e.Killer.Team
			if killer_team == 2 {
				current_state+="t_kill "
			} else if killer_team == 3 {
				current_state+="ct_kill "
			}
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
	
	p.RegisterEventHandler(func(e events.ItemPickup) {
		
		if !(e.Player == nil){
			team_equip := e.Player.Team
			team_name := ""
			if team_equip == 2 {
				team_name="t_"
			} else if team_equip == 3 {
				team_name="ct_"
			}
			weapon_equipped := strings.ReplaceAll(e.Weapon.Weapon.String()," ","_")
			weapon_equipped=strings.ReplaceAll(weapon_equipped,"-","_")
			weapon_equipped=strings.ToLower(weapon_equipped)+" "
			current_state+=team_name+weapon_equipped
		}
		
	})
	
	p.RegisterEventHandler(func(e events.RoundStart) {
		gs:= p.GameState()
		if !(gs == nil) {
			if gs.TeamCounterTerrorists().Score == 0 && gs.TeamTerrorists().Score == 0 && !game_started {
				current_state= map_name + " "
				game_reset=true
			} else if gs.TeamCounterTerrorists().Score == 0 && gs.TeamTerrorists().Score == 0 {
				current_state+= "match_end "
			}
			if gs.TeamCounterTerrorists().Score+gs.TeamTerrorists().Score > 10 && game_reset {
				game_started=true
			}
		}
		current_state+="round_start ct_"+strconv.Itoa(gs.TeamCounterTerrorists().Score) + " t_" + strconv.Itoa(gs.TeamTerrorists().Score) + " " 
	})
	
//	p.RegisterEventHandler(func(e events.PlayerSpottersChanged) {
//		if !(e.Spotted == nil){
//			processSpotEvent(e.Spotted)
//		}
		
//	})
	
//	p.RegisterEventHandler(func(e events.ScoreUpdated) {
//		if !(e.TeamState == nil){
//			if e.NewScore == 0 && e.TeamState.Opponent.Score == 0 {
//				current_state=map_name + " "
//			}
//		}
//		
//	})


	//p.RegisterEventHandler(func(e events.FrameDone) {
	//	gs := p.GameState()
	//	if !(gs == nil){
	//		processFrameEnd(gs)
	//	}
		
	//})
	
	
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
	if current_state[0:3]!="de_" {
		current_state = map_name + " " + current_state
	}
	_, err = file_write.WriteString(current_state)
	checkError(err)
	// Parse to end
}

//func processFrameEnd(gs *IGameState){
//	tr := gs.TeamTerrorists()
//	ct := gs.TeamCounterTerrorists()
//	processPlayerPositions(tr,ct)
//}

//func processPlayerPositions(tr *common.TeamState, ct *common.TeamState){
//	tr_members = tr.Members()
//	ct_members = ct.Members()
//	var playerId = 1
//	for _, tr_player := range tr_members {
//		switch playerId
//	}
//}

func processSpotEvent(player *common.Player){
	if !(player.TeamState == nil){
	
		enemy_team := player.TeamState.Opponent.Members()
		for _,enemy_player := range enemy_team {
			if player.IsSpottedBy(enemy_player){
				if player.Team == 2 {
					current_state+="t_player_spotted "
				} else if player.Team == 3 {
					current_state+="ct_player_spotted "
				}
				
			}
		}	
	}
}

// Run like this: go run print_events.go -demo /path/to/demo.dem
func main() {
	
	dem_path:= os.Args[1]
	file_id_str:=os.Args[2]
	dest_dir:=os.Args[3]
	file_id, err := strconv.Atoi(file_id_str)
	checkError(err)
	processDemoFile(dem_path,file_id,dest_dir)
}


func checkError(err error) {
	if err != nil {
		print("error!")
		panic(err)
	}
}
