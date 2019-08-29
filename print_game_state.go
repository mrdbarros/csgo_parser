package main

import (
	"fmt"
	"os"
	"strconv"
	dem "github.com/markus-wa/demoinfocs-golang"
	r3 "github.com/golang/geo/r3"
	//ex "github.com/markus-wa/demoinfocs-golang/examples"
	events "github.com/markus-wa/demoinfocs-golang/events"
	common "github.com/markus-wa/demoinfocs-golang/common"
	"strings"
//	"time"
)

var current_state=""
var game_reset = false
var game_started = false
var discretize_factor = 20.0
var tr_pos = make(map[int]r3.Vector)
var ct_pos = make(map[int]r3.Vector)
var round_start_time int
var last_update =0
var last_time_event =0
var tick_rate=0
var pos_update_interval=2

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
	var ct_1_pos r3.Vector
	var ct_2_pos r3.Vector
	var ct_3_pos r3.Vector
	var ct_4_pos r3.Vector
	var ct_5_pos r3.Vector
	ct_pos = map[int]r3.Vector{
		1: ct_1_pos,
		2: ct_2_pos,
		3: ct_3_pos,
		4: ct_4_pos,
		5: ct_5_pos,
	}
	var tr_1_pos r3.Vector
	var tr_2_pos r3.Vector
	var tr_3_pos r3.Vector
	var tr_4_pos r3.Vector
	var tr_5_pos r3.Vector
	tr_pos = map[int]r3.Vector{
		1: tr_1_pos,
		2: tr_2_pos,
		3: tr_3_pos,
		4: tr_4_pos,
		5: tr_5_pos,
	}
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
		new_score:="ct_"+strconv.Itoa(gs.TeamCounterTerrorists().Score) + " t_" + strconv.Itoa(gs.TeamTerrorists().Score) + " " 
		//print(new_score)
		current_state+="round_start "+new_score
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


	p.RegisterEventHandler(func(e events.FrameDone) {
		gs := p.GameState()
		if !(gs == nil){
			processFrameEnd(gs,p)
		}
		
	})
	
	p.RegisterEventHandler(func(e events.SmokeStart) {
		ge := e.GrenadeEvent
		pos := ge.Position
		discrete_pos := discretizePos(pos)
		current_state+="smoke_start "+formatPosForPrint(discrete_pos)
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
	
	p.RegisterEventHandler(func(e events.RoundFreezetimeEnd) {
		round_start_time = p.CurrentFrame()/tick_rate
		current_state+="freeze_time_end "
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

func processFrameEnd(gs dem.IGameState,p dem.IParser){
	//print(p.Header().PlaybackFrames)
	if getRoundTime(p)%pos_update_interval == 0 && getCurrentTime(p)!=last_update {
		last_update = getCurrentTime(p)
		tr := gs.TeamTerrorists()
		ct := gs.TeamCounterTerrorists()
		processPlayerPositions(tr,ct)
	}
	if getRoundTime(p)%(pos_update_interval*2) == 0 && getRoundTime(p)!=last_time_event {
		last_time_event = getRoundTime(p)
		current_state+="time_event_"+strconv.Itoa(last_time_event)+" "
	}
}

func getRoundTime(p dem.IParser)(int){
	return int(getCurrentTime(p)-round_start_time)
}

func getCurrentTime(p dem.IParser)(int){
	return p.CurrentFrame()/tick_rate
}

func processPlayerPositions(tr *common.TeamState, ct *common.TeamState){
	updateDiscretePositions(tr)
	updateDiscretePositions(ct)
}

func formatPosForPrint(pos r3.Vector)(string){
	print_pos := discretizePos(pos)
	return "x_coor "+strconv.Itoa(int(print_pos.X))+" y_coor "+strconv.Itoa(int(print_pos.Y))+" z_coor "+strconv.Itoa(int(print_pos.Z))+" "

}

func updateDiscretePositions(team *common.TeamState){
	playerId:=1
	for _, player := range team.Members() {
		if team.Team() == 2 {
			if !discretizePos(tr_pos[playerId]).ApproxEqual(discretizePos(player.Position)) {
				tr_pos[playerId]=player.Position
				current_state+="tr_" + strconv.Itoa(playerId)+ "_pos "+formatPosForPrint(player.Position)
			}
		} else if team.Team() == 3 {
			if !discretizePos(ct_pos[playerId]).ApproxEqual(discretizePos(player.Position)) {
				ct_pos[playerId]=player.Position
				current_state+="ct_"+strconv.Itoa(playerId)+"_pos "+formatPosForPrint(player.Position)
			}
		}
		playerId+=1
	}
}


func discretizePos(pos r3.Vector)(disc_pos_ret r3.Vector){
	var disc_pos r3.Vector
	disc_pos.X = pos.X/discretize_factor
	disc_pos.Y = pos.Y/discretize_factor
	disc_pos.Z = pos.Z/discretize_factor
	return disc_pos
}
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
	tick_rate,err=strconv.Atoi(os.Args[4])
	checkError(err)	
	processDemoFile(dem_path,file_id,dest_dir)
}


func checkError(err error) {
	if err != nil {
		print("error!")
		panic(err)
	}
}
