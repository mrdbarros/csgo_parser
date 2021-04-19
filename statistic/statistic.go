package statistic

type PlayersStatistics struct {
	StatisticsName       []string `json:"statName"`
	PlayerId    []uint64 `json:"playerId"`
	PlayerName []string `json:"playerName"`
	StatisticValue []float64 `json:"statValue"`
}



