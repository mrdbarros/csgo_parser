package service

import (
	"context"

	"github.com/go-kit/kit/endpoint"
)

type Endpoints struct {
	GetStatistics endpoint.Endpoint

}

func MakeEndpoints(s Service) Endpoints {
	return Endpoints{
		GetStatistics: makeGetStatisticsEndpoint(s),
	}
}

func makeGetStatisticsEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetStatisticsRequest)
		statistics, err := s.GetStatistics(
			ctx, 
			req.Statistics,
			req.Tournaments,
			req.Matches,
			req.Players,
			req.StartDate,
			req.EndDate,
		)
		return GetStatisticsResponse{
			StatisticName: statistics.StatisticsName,
			PlayerId: statistics.PlayerId,
			PlayerName: statistics.PlayerName,
			StatValue: statistics.StatisticValue,
		}, err
	}
}


