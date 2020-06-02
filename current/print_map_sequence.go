package main

import (
	"fmt"
	"os"

	dem "github.com/markus-wa/demoinfocs-golang/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/pkg/demoinfocs/events"
)

func main() {
	f, err := os.Open("/home/marcel/projetos/data/csgo_analyze/replays/g2-vs-faze-m2-train.dem")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	p := dem.NewParser(f)

	// Register handler on kill events
	p.RegisterEventHandler(func(e events.Kill) {
		var hs string
		if e.IsHeadshot {
			hs = " (HS)"
		}
		var wallBang string
		if e.PenetratedObjects > 0 {
			wallBang = " (WB)"
		}
		fmt.Printf("%s <%v%s%s> %s\n", e.Killer, e.Weapon, hs, wallBang, e.Victim)
	})

	// Parse to end
	err = p.ParseToEnd()
	if err != nil {
		panic(err)
	}
}
