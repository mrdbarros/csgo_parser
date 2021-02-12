package composite_handlers

import events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"

type ADRCalculator struct {
	statisticHolder
}

func (kc *ADRCalculator) Register(bh *BasicHandler) error {
	kc.basicHandler = bh
	bh.RegisterPlayerHurtSubscriber(interface{}(kc).(PlayerHurtSubscriber))
	bh.RegisterRoundFreezetimeEndSubscriber(interface{}(kc).(RoundFreezetimeEndSubscriber))
	kc.baseStatsHeaders = []string{"Total Damage Done", "Total Damage Done_T", "Total Damage Done_CT"}
	kc.defaultValues = make(map[string]float64)
	return nil
}

func (kc *ADRCalculator) RoundFreezetimeEndHandler(e events.RoundFreezetimeEnd) {
	if kc.basicHandler.roundNumber-1 < len(kc.playerStats) {
		kc.playerStats = kc.playerStats[:kc.basicHandler.roundNumber-1]
	}
	kc.AddNewRound()
}

func (kc *ADRCalculator) PlayerHurtHandler(e events.PlayerHurt) {

	if e.Attacker != nil {
		var addAmmount float64
		if e.Attacker.Team != e.Player.Team {
			addAmmount = float64(e.HealthDamageTaken)
		} else {
			addAmmount = -float64(e.HealthDamageTaken)
		}
		kc.addToPlayerStat(e.Attacker, addAmmount, "Total Damage Done")
	}

}
