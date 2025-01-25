package db

import (
	"github.com/glebarez/sqlite"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/schema"
	"gorm.io/gorm"
)

func NewDB() (*gorm.DB, error) {
	database, err := gorm.Open(sqlite.Open("client.sqlite3"), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := database.DB()
	sqlDB.Exec("PRAGMA foreign_keys = ON")

	err = database.AutoMigrate(&schema.File{}, &schema.Chunk{}, &schema.Peer{}, &schema.PeerChunk{})

	if err != nil {
		return nil, err
	}
	return database, nil
}
