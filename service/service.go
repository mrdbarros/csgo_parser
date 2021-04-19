package service

import (
	"context"
	"errors"
	"time"

	"github.com/go-kit/kit/log"
	database "github.com/mrdbarros/csgo_analyze/database"
	statistic "github.com/mrdbarros/csgo_analyze/statistic"
)

type Service interface {
	GetStatistics(
		ctx context.Context, 
		stats []string,
		tournaments []int,
		matches []int,
		players []uint64,
		startDate time.Time,
		endDate time.Time) (statistic.PlayersStatistics, error)
		
}

type Repository interface {
	GetStatistics(
		ctx context.Context, 
		stats []string,
		Tournaments []int,
		Matches []int,
		Players []uint64,
		StartDate time.Time,
		EndDate time.Time,
		) (statistic.PlayersStatistics,error)
}


var RepoErr = errors.New("Unable to handle Repo Request")

type repo struct {
	database.Database
	logger log.Logger
}

func NewRepo(db database.Database, logger log.Logger) Repository {
	return &repo{
		Database: db,
		logger:logger,
	}
}