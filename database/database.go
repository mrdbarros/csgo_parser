package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"

	statistic "github.com/mrdbarros/csgo_analyze/statistic"
	utils "github.com/mrdbarros/csgo_analyze/utils"
)

type Database struct {
	dbConn *sql.DB
}

func (db Database) InsertPlayer(playerID uint64, playerName string) {
	insForm, err := db.dbConn.Prepare("INSERT INTO PLAYER(idPLAYER, NAME) VALUES(?,?) ON DUPLICATE KEY UPDATE NAME=?")
	utils.CheckError(err)
	insForm.Exec(playerID, playerName, playerName)
	insForm.Close()
}

func (db Database) CheckIfProcessed( demFileHash string) bool {

	sqlResult, err := db.dbConn.Query("SELECT idCSGO_MATCH FROM CSGO_MATCH WHERE DEMO_FILE_HASH=?", demFileHash)
	utils.CheckError(err)
	if sqlResult.Next() {
		sqlResult.Close()
		return true
	} else {
		sqlResult.Close()
		return false
	}

}

func (db Database) InsertMatch( fileName string, demFileHash string, mapName string, terroristFirstTeamScore int, ctFirstTeamScore int,
	matchDateTime time.Time, overwriteMatch bool) (matchID int) {
 	
	var	insForm *sql.Stmt
	var err error
	dt := matchDateTime.Format(time.RFC3339)
	if !db.CheckIfProcessed(demFileHash){
		insForm, err = db.dbConn.Prepare("INSERT INTO CSGO_MATCH(SCORE_FIRST_T,SCORE_FIRST_CT,MAP,MATCH_DATETIME,DEMO_FILE_HASH,FILE_NAME) VALUES(?,?,?,?,?,?)")
		utils.CheckError(err)
		_, err = insForm.Exec(terroristFirstTeamScore, ctFirstTeamScore, mapName, dt,demFileHash, fileName)
		utils.CheckError(err)
	} else {
		insForm, err = db.dbConn.Prepare("UPDATE CSGO_MATCH SET SCORE_FIRST_T=?, SCORE_FIRST_CT=?, MAP=?, MATCH_DATETIME=?, FILE_NAME=? WHERE DEMO_FILE_HASH = ?")
		utils.CheckError(err)
		_, err = insForm.Exec(terroristFirstTeamScore, ctFirstTeamScore, mapName, dt, fileName,demFileHash)
		utils.CheckError(err)
		
	}
	
	
	insForm.Close()
	if err != nil {
		if err.(*mysql.MySQLError).Number == 1062 {
			fmt.Println("Demo file already in database")
			if overwriteMatch {
				fmt.Println("Overwrite mode on. Deleting old data.")
				insForm, err = db.dbConn.Prepare("DELETE STATISTICS_PLAYER_MATCH_FACT FROM STATISTICS_PLAYER_MATCH_FACT INNER JOIN " +
					"CSGO_MATCH ON STATISTICS_PLAYER_MATCH_FACT.idCSGO_MATCH = CSGO_MATCH.idCSGO_MATCH WHERE CSGO_MATCH.DEMO_FILE_HASH = ?")
				_, err = insForm.Exec(demFileHash)
				insForm.Close()
				utils.CheckError(err)
			} else {
				utils.CheckError(err)
			}
		}
	}

	sqlResult, err := db.dbConn.Query("SELECT idCSGO_MATCH FROM CSGO_MATCH WHERE DEMO_FILE_HASH=?", demFileHash)
	utils.CheckError(err)
	sqlResult.Next()
	sqlResult.Scan(&matchID)
	sqlResult.Close()
	return matchID
}

func OpenDBConn() Database {
	db, err := sql.Open("mysql", "marcel:basecsteste1!@tcp(127.0.0.1:3306)/CSGO_ANALYTICS")

	utils.CheckError(err)
	return Database{dbConn:db}
}

func (db Database) InsertBaseStatistics( tempHeader []string) (statIds []int) {
	var newID int
	for _, statName := range tempHeader {
		insForm, err := db.dbConn.Prepare("INSERT IGNORE INTO BASE_STATISTIC(NAME) VALUES(?)")
		utils.CheckError(err)
		insForm.Exec(statName)
		insForm.Close()
		sqlResult, err := db.dbConn.Query("SELECT idBASE_STATISTIC FROM BASE_STATISTIC WHERE NAME=?", statName)
		utils.CheckError(err)
		sqlResult.Next()
		sqlResult.Scan(&newID)
		sqlResult.Close()
		statIds = append(statIds, newID)

	}
	return statIds
}

func (db Database) InsertStatisticsFacts( statIDs []int, tempData []float64, playerID uint64, matchID int) {
	for i, statID := range statIDs {
		insForm, err := db.dbConn.Prepare("INSERT INTO STATISTICS_PLAYER_MATCH_FACT(idCSGO_MATCH,idPLAYER,idBASE_STATISTIC,VALUE) VALUES(?,?,?,?)")
		utils.CheckError(err)
		insForm.Exec(matchID, playerID, statID, tempData[i])
		insForm.Close()
	}
}



func (db Database) GetStatistics(
		ctx context.Context, 
		stats []string,
		Tournaments []int,
		Matches []int,
		Players []uint64,
		StartDate time.Time,
		EndDate time.Time,
		) (statistic.PlayersStatistics,error) {

			sqlResult, err := db.dbConn.Query(`SELECT PLAYER.idPLAYER AS ID_PLAYER,PLAYER.NAME AS PLAYER_NAME,BASE_STATISTIC.NAME AS STATISTIC_NAME,SUM(STATISTICS_PLAYER_MATCH_FACT.VALUE)  AS VALUE \
				FROM ((STATISTICS_PLAYER_MATCH_FACT INNER JOIN PLAYER ON PLAYER.idPLAYER = STATISTICS_PLAYER_MATCH_FACT.idPLAYER ) 
				INNER JOIN BASE_STATISTIC ON BASE_STATISTIC.idBASE_STATISTIC = STATISTICS_PLAYER_MATCH_FACT.idBASE_STATISTIC)
				INNER JOIN CSGO_MATCH ON  STATISTICS_PLAYER_MATCH_FACT.idCSGO_MATCH = CSGO_MATCH.idCSGO_MATCH
				WHERE PLAYER.idPLAYER IN (?" + strings.Repeat(",?", len(Players)-1) + ")
				AND CSGO_MATCH.MATCH_DATETIME between ? and ?
				GROUP BY PLAYER.idPLAYER,PLAYER.NAME, BASE_STATISTIC.NAME

				UNION ALL


				SELECT T1.idPLAYER,PLAYER_NAME,STATISTIC_NAME, IF(DENOMINATOR_VALUE!=0,NUMERATOR_VALUE/DENOMINATOR_VALUE,0) AS VALUE FROM
				(SELECT PLAYER.idPLAYER,PLAYER.NAME AS PLAYER_NAME,RATIO_STATISTIC.NAME as STATISTIC_NAME,RATIO_STATISTIC.NUMERATOR,RATIO_STATISTIC.DENOMINATOR, 
					SUM(STATISTICS_PLAYER_MATCH_FACT.VALUE) AS NUMERATOR_VALUE FROM 
				(STATISTICS_PLAYER_MATCH_FACT 
					INNER JOIN RATIO_STATISTIC ON RATIO_STATISTIC.NUMERATOR = STATISTICS_PLAYER_MATCH_FACT.idBASE_STATISTIC)
						INNER JOIN PLAYER ON PLAYER.idPLAYER = STATISTICS_PLAYER_MATCH_FACT.idPLAYER
						INNER JOIN CSGO_MATCH ON  STATISTICS_PLAYER_MATCH_FACT.idCSGO_MATCH = CSGO_MATCH.idCSGO_MATCH
						WHERE PLAYER.idPLAYER IN (?" + strings.Repeat(",?", len(Players)-1) + ")
						AND CSGO_MATCH.MATCH_DATETIME between ? and ?
						GROUP BY PLAYER.idPLAYER, RATIO_STATISTIC.NUMERATOR,RATIO_STATISTIC.DENOMINATOR) T1
				INNER JOIN
				(SELECT PLAYER.idPLAYER,RATIO_STATISTIC.NUMERATOR,RATIO_STATISTIC.DENOMINATOR, 
					SUM(STATISTICS_PLAYER_MATCH_FACT.VALUE) AS DENOMINATOR_VALUE FROM 
				(STATISTICS_PLAYER_MATCH_FACT 
					INNER JOIN RATIO_STATISTIC ON RATIO_STATISTIC.DENOMINATOR = STATISTICS_PLAYER_MATCH_FACT.idBASE_STATISTIC)
						INNER JOIN PLAYER ON PLAYER.idPLAYER = STATISTICS_PLAYER_MATCH_FACT.idPLAYER
						INNER JOIN CSGO_MATCH ON  STATISTICS_PLAYER_MATCH_FACT.idCSGO_MATCH = CSGO_MATCH.idCSGO_MATCH
						WHERE PLAYER.idPLAYER IN (?" + strings.Repeat(",?", len(Players)-1) + ")
						AND CSGO_MATCH.MATCH_DATETIME between ? and ?
						GROUP BY PLAYER.idPLAYER, RATIO_STATISTIC.NUMERATOR,RATIO_STATISTIC.DENOMINATOR) T2 ON T1.idPLAYER = T2.idPLAYER AND T1.NUMERATOR = T2.NUMERATOR AND T1.DENOMINATOR=T2.DENOMINATOR

				UNION ALL

				SELECT PLAYER.idPLAYER, PLAYER.NAME,'Matches' AS STATISTIC_NAME,COUNT(DISTINCT(CSGO_MATCH.DEMO_FILE_HASH)) AS VALUE
					FROM (PLAYER
						INNER JOIN STATISTICS_PLAYER_MATCH_FACT ON STATISTICS_PLAYER_MATCH_FACT.idPLAYER = PLAYER.idPLAYER)
						INNER JOIN CSGO_MATCH ON CSGO_MATCH.idCSGO_MATCH = STATISTICS_PLAYER_MATCH_FACT.idCSGO_MATCH
					WHERE PLAYER.idPLAYER IN (?" + strings.Repeat(",?", len(Players)-1) + ")
						AND CSGO_MATCH.MATCH_DATETIME between ? and ?
						GROUP BY PLAYER.idPLAYER,PLAYER.NAME,STATISTIC_NAME`,Players,StartDate,EndDate,Players,StartDate,EndDate,Players,StartDate,EndDate,Players,StartDate,EndDate)

			var playersStats statistic.PlayersStatistics
			var statName string
			var playerId uint64
			var playerName string
			var statValue float64
			utils.CheckError(err)
			sqlResult.Next()
			sqlResult.Scan(&playerId,&playerName,&statName,&statValue)
			sqlResult.Close()
			playersStats.PlayerId = append(playersStats.PlayerId,playerId)
			playersStats.PlayerName = append(playersStats.PlayerName,playerName)
			playersStats.StatisticsName = append(playersStats.StatisticsName,statName)
			playersStats.StatisticValue = append(playersStats.StatisticValue,statValue)
			return playersStats,err

}





func (db Database) Close() error {
	return db.dbConn.Close()
}