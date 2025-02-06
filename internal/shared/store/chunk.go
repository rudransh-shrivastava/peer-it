package store

import (
	"github.com/rudransh-shrivastava/peer-it/internal/shared/schema"
	"gorm.io/gorm"
)

type ChunkStore struct {
	DB *gorm.DB
}

func NewChunkStore(db *gorm.DB) *ChunkStore {
	return &ChunkStore{DB: db}
}

func (cs *ChunkStore) CreateChunk(file *schema.File, size int, index int, hash string, withMetadata bool) error {
	chunk := schema.Chunk{
		File:  *(file),
		Index: index,
	}
	if withMetadata {
		chunkMetadata := schema.ChunkMetadata{
			Chunk:     chunk,
			ChunkSize: size,
			ChunkHash: hash,
		}
		err := cs.DB.Create(&chunkMetadata).Error
		if err != nil {
			return err
		}
	}
	err := cs.DB.Create(&chunk).Error
	if err != nil {
		return err
	}
	return nil
}

func (cs *ChunkStore) GetChunks(fileHash string) (*[]schema.Chunk, error) {
	file := &schema.File{}
	err := cs.DB.First(&file, "hash = ?", fileHash).Error
	chunks := []schema.Chunk{}
	err = cs.DB.Find(&chunks, "file_id = ?", file.ID).Error
	if err != nil {
		return nil, err
	}
	return &chunks, nil
}
