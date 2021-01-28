package composite_handlers

import events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"

type ADRCalculator struct {
	statisticHolder
}

func (kc *ADRCalculator) Register(bh *BasicHandler) error {
	kc.basicHandler = bh
	bh.RegisterRoundStartSubscriber(interface{}(kc).(RoundStartSubscriber))
	bh.RegisterPlayerHurtSubscriber(interface{}(kc).(PlayerHurtSubscriber))
	bh.RegisterRoundFreezetimeEndSubscriber(interface{}(kc).(RoundFreezetimeEndSubscriber))
	kc.baseStatsHeaders = []string{"Total Damage Done", "Total Damage Done_T", "Total Damage Done_CT"}
	kc.ratioStats = append(kc.ratioStats, [3]string{"ADR", "Total Damage Done", "Rounds"})
	kc.defaultValues = make(map[string]float64)
	return nil
}

func (kc *ADRCalculator) RoundStartHandler(e events.RoundStart) {
	if kc.basicHandler.roundNumber-1 < len(kc.playerStats) {
		kc.playerStats = kc.playerStats[:kc.basicHandler.roundNumber-1]
	}
}

func (kc *ADRCalculator) RoundFreezetimeEndHandler(e events.RoundFreezetimeEnd) {

	kc.AddNewRound()
}

func (kc *ADRCalculator) PlayerHurtHandler(e events.PlayerHurt) {

	if e.Attacker != nil {
		kc.addToPlayerStat(e.Attacker, float64(e.HealthDamageTaken), "Total Damage Done")
	}

}
