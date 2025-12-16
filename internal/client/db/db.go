package db

import (
	"database/sql"

	peerdb "github.com/rudransh-shrivastava/peer-it/internal/db"
)

func NewDB(id string) (*sql.DB, error) {
	return peerdb.Open("client-" + id + ".sqlite3")
}
