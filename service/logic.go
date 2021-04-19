package service

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	statistic "github.com/mrdbarros/csgo_analyze/statistic"
)

type service struct {
	repository Repository
	logger    log.Logger
}

func NewService(rep Repository, logger log.Logger) Service {
	return &service{
		repository: rep,
		logger:    logger,
	}
}

func (s service) GetStatistics(ctx context.Context, 
		stats []string,
		tournaments []int,
		matches []int,
		players []uint64,
		startDate time.Time,
		endDate time.Time) (statistic.PlayersStatistics, error) {
	logger := log.With(s.logger, "method", "GetStatistics")
	
	
	playersStats,err := s.repository.GetStatistics(ctx, stats,tournaments,matches,players,startDate,endDate)
	if err != nil {
		level.Error(logger).Log("err", err)
		return statistic.PlayersStatistics{}, err
	}

	logger.Log("got statistics")

	return playersStats, nil
}

