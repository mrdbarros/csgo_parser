package main

import (
	"fmt"
	"os"
	"io/ioutil"
	dem "github.com/markus-wa/demoinfocs-golang"
	//ex "github.com/markus-wa/demoinfocs-golang/examples"
	events "github.com/markus-wa/demoinfocs-golang/events"
	"strings"
)

func processDemoFile(demPath string){
	f, err := os.Open(demPath)
	defer f.Close()
	checkError(err)

	p := dem.NewParser(f)

	// Parse header
	header, err := p.ParseHeader()
	checkError(err)
	fmt.Println("Map:", header.MapName)
	
	
	p.RegisterEventHandler(func(e events.Kill) {
		killer_team := e.Killer.Team
		if killer_team == 2 {
			fmt.Printf("t_kill ")
		} else if killer_team == 3 {
			fmt.Printf("ct_kill ")
		}
	})
	
	p.RegisterEventHandler(func(e events.RoundEnd) {
		
		win_team := e.Winner
		if win_team == 2 {
			fmt.Printf("t_round_win ")
		} else if win_team == 3 {
			fmt.Printf("ct_round_win ")
		} else {
			fmt.Printf("invalid_round_end ")
		}
	})
	
	p.RegisterEventHandler(func(e events.RoundStart) {
		

		fmt.Printf("round_start ")
	})
	
	p.RegisterEventHandler(func(e events.RoundEndOfficial) {
		

		fmt.Printf("round_end_official ")
	})
	
	p.RegisterEventHandler(func(e events.BombPlanted) {
		

		fmt.Printf("bomb_planted ")
	})
	
	p.RegisterEventHandler(func(e events.BombPlantBegin) {
		

		fmt.Printf("bomb_plant_begin ")
	})
	
	
	
	err = p.ParseToEnd()
	checkError(err)


	// Parse to end
}

// Run like this: go run print_events.go -demo /path/to/demo.dem
func main() {
	demSetPath:="C:/Users/marcel.barros/go/src/github.com/markus-wa/demoinfocs-golang/data"

	dems, err := ioutil.ReadDir(demSetPath)
	checkError(err)
	for _, d := range dems {
		name := d.Name()
		if strings.HasSuffix(name, ".dem") {
			fmt.Printf("Parsing '%s/%s'\n", demSetPath, name)
			processDemoFile(demSetPath+"/"+name)
	
		}
	}
}


func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
