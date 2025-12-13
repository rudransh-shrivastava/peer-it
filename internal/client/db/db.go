package db

import (
	"database/sql"

	peerdb "github.com/rudransh-shrivastava/peer-it/internal/db"
)

func NewDB(index string) (*sql.DB, error) {
	return peerdb.Open("client-" + index + ".sqlite3")
}
