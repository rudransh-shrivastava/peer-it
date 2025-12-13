package db

import (
	"github.com/glebarez/sqlite"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/schema"
	"gorm.io/gorm"
)

func NewDB(index string) (*gorm.DB, error) {
	dbName := "client-" + index + ".sqlite3"
	database, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := database.DB()
	if err != nil {
		return nil, err
	}

	if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, err
	}

	if err := database.AutoMigrate(&schema.File{}, &schema.Chunk{}, &schema.Peer{}, &schema.Swarm{}); err != nil {
		return nil, err
	}

	return database, nil
}
