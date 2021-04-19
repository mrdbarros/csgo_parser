package service

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type (
	GetStatisticsRequest struct {
		Statistics []string `json:"stats"`
		Tournaments []int `json:"tournaments"`
		Matches []int `json:"matches"`
		Players []uint64 `json:"players"`
		StartDate time.Time `json:"startDate"`
		EndDate time.Time `json:"endDate"`

	}
	GetStatisticsResponse struct {
		StatisticName []string `json:"statName"`
		PlayerId     []uint64 `json:"playerId"`
		PlayerName	[]string `json:"playerName"`
		StatValue  []float64 `json:"statValue"`
	}


)


func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	return json.NewEncoder(w).Encode(response)
}

func decodeUserReq(ctx context.Context, r *http.Request) (interface{}, error) {
	var req GetStatisticsRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}
	return req, nil
}


