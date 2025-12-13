package db

import (
	"database/sql"

	peerdb "github.com/rudransh-shrivastava/peer-it/internal/db"
)

func NewDB() (*sql.DB, error) {
	return peerdb.Open("tracker.sqlite3")
}
