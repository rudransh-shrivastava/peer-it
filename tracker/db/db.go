package db

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type File struct {
	ID           uint `gorm:"primaryKey"`
	Name         string
	Size         int
	MaxChunkSize int
	TotalChunks  int
	Checksum     string
	CreatedAt    int64
}

type Chunk struct {
	ID       uint `gorm:"primaryKey"`
	Index    int
	FileID   uint `gorm:"not null;foreignKey:FileID;constraint:OnDelete:CASCADE"`
	File     File `gorm:"constraint:OnDelete:CASCADE"`
	Size     int
	Checksum string
}

type Client struct {
	ID          uint `gorm:"primaryKey"`
	IPAddress   string
	Port        int
	ConnectedAt int64
}

type ClientChunk struct {
	ID       uint   `gorm:"primaryKey"`
	ClientID uint   `gorm:"not null;foreignKey:ClientID;constraint:OnDelete:CASCADE"`
	Client   Client `gorm:"constraint:OnDelete:CASCADE"`
	ChunkID  uint   `gorm:"not null;foreignKey:ChunkID;constraint:OnDelete:CASCADE"`
	Chunk    Chunk  `gorm:"constraint:OnDelete:CASCADE"`
}

func NewDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("db.sqlite3"), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	sqlDB.Exec("PRAGMA foreign_keys = ON")

	err = db.AutoMigrate(&File{}, &Chunk{}, &Client{}, &ClientChunk{})

	if err != nil {
		return nil, err
	}
	return db, nil
}
