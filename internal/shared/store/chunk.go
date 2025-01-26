package store

import (
	"fmt"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/schema"
	"gorm.io/gorm"
)

type ChunkStore struct {
	DB *gorm.DB
}

func NewChunkStore(db *gorm.DB) *ChunkStore {
	return &ChunkStore{DB: db}
}

func (cs *ChunkStore) CreateChunk(file *schema.File, size int, index int, checksum string) {
	chunk := schema.Chunk{
		File:     *(file),
		Size:     size,
		Index:    index,
		Checksum: checksum,
	}

	fmt.Println("creating the chunks ----++++=----", index)
	cs.DB.Create(&chunk)
}
