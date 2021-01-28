package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"

	utils "github.com/mrdbarros/csgo_analyze/utils"
)

func InsertPlayer(dbConn *sql.DB, playerID uint64, playerName string) {
	insForm, err := dbConn.Prepare("INSERT INTO PLAYER(idPLAYER, NAME) VALUES(?,?) ON DUPLICATE KEY UPDATE NAME=?")
	utils.CheckError(err)
	insForm.Exec(playerID, playerName, playerName)
	insForm.Close()
}

func InsertMatch(dbConn *sql.DB, fileName string, demFileHash string, mapName string, terroristFirstTeamScore int, ctFirstTeamScore int,
	matchDateTime time.Time, overwriteMatch bool) (matchID int) {

	dt := matchDateTime.Format(time.RFC3339)
	insForm, err := dbConn.Prepare("INSERT INTO CSGO_MATCH(SCORE_FIRST_T,SCORE_FIRST_CT,MAP,MATCH_DATETIME,DEMO_FILE_HASH,FILE_NAME) VALUES(?,?,?,?,?,?)")
	utils.CheckError(err)
	_, err = insForm.Exec(terroristFirstTeamScore, ctFirstTeamScore, mapName, dt, demFileHash, fileName)
	insForm.Close()
	if err != nil {
		if err.(*mysql.MySQLError).Number == 1062 {
			fmt.Println("Demo file already in database")
			if overwriteMatch {
				fmt.Println("Overwrite mode on. Deleting old data.")
				insForm, err = dbConn.Prepare("DELETE STATISTICS_PLAYER_MATCH_FACT FROM STATISTICS_PLAYER_MATCH_FACT INNER JOIN " +
					"CSGO_MATCH ON STATISTICS_PLAYER_MATCH_FACT.idCSGO_MATCH = CSGO_MATCH.idCSGO_MATCH WHERE CSGO_MATCH.DEMO_FILE_HASH = ?")
				_, err = insForm.Exec(demFileHash)
				insForm.Close()
				utils.CheckError(err)
			} else {
				utils.CheckError(err)
			}
		}
	}

	sqlResult, err := dbConn.Query("SELECT idCSGO_MATCH FROM CSGO_MATCH WHERE DEMO_FILE_HASH=?", demFileHash)
	utils.CheckError(err)
	sqlResult.Next()
	sqlResult.Scan(&matchID)
	sqlResult.Close()
	return matchID
}

func OpenDBConn() *sql.DB {
	db, err := sql.Open("mysql", "marcel:basecsteste1!@tcp(127.0.0.1:3306)/CSGO_ANALYTICS")

	utils.CheckError(err)
	return db
}

func InsertBaseStatistics(dbConn *sql.DB, tempHeader []string) (statIds []int) {
	var newID int
	for _, statName := range tempHeader {
		insForm, err := dbConn.Prepare("INSERT IGNORE INTO BASE_STATISTIC(NAME) VALUES(?)")
		utils.CheckError(err)
		insForm.Exec(statName)
		insForm.Close()
		sqlResult, err := dbConn.Query("SELECT idBASE_STATISTIC FROM BASE_STATISTIC WHERE NAME=?", statName)
		utils.CheckError(err)
		sqlResult.Next()
		sqlResult.Scan(&newID)
		sqlResult.Close()
		statIds = append(statIds, newID)

	}
	return statIds
}

func InsertStatisticsFacts(dbConn *sql.DB, statIDs []int, tempData []float64, playerID uint64, matchID int) {
	for i, statID := range statIDs {
		insForm, err := dbConn.Prepare("INSERT INTO STATISTICS_PLAYER_MATCH_FACT(idCSGO_MATCH,idPLAYER,idBASE_STATISTIC,VALUE) VALUES(?,?,?,?)")
		utils.CheckError(err)
		insForm.Exec(matchID, playerID, statID, tempData[i])
		insForm.Close()
	}
}
