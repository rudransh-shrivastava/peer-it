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

func (cs *ChunkStore) CreateChunk(file *schema.File, size int, index int, hash string, isAvailable bool) error {
	chunk := schema.Chunk{
		ChunkHash:   hash,
		ChunkIndex:  index,
		ChunkSize:   size,
		File:        *file,
		IsAvailable: isAvailable,
	}
	return cs.DB.Create(&chunk).Error
}

func (cs *ChunkStore) GetChunk(fileHash string, chunkIndex int) (*schema.Chunk, error) {
	file := &schema.File{}
	if err := cs.DB.First(&file, "hash = ?", fileHash).Error; err != nil {
		return nil, err
	}

	chunk := schema.Chunk{}
	if err := cs.DB.First(&chunk, "file_id = ? AND chunk_index = ?", file.ID, chunkIndex).Error; err != nil {
		return nil, err
	}

	return &chunk, nil
}

func (cs *ChunkStore) GetChunks(fileHash string) (*[]schema.Chunk, error) {
	file := &schema.File{}
	if err := cs.DB.First(&file, "hash = ?", fileHash).Error; err != nil {
		return nil, err
	}

	chunks := []schema.Chunk{}
	if err := cs.DB.Find(&chunks, "file_id = ?", file.ID).Error; err != nil {
		return nil, err
	}

	return &chunks, nil
}

func (cs *ChunkStore) MarkChunkAvailable(fileHash string, chunkIndex int) error {
	file := &schema.File{}
	if err := cs.DB.First(&file, "hash = ?", fileHash).Error; err != nil {
		return err
	}

	chunk := schema.Chunk{}
	if err := cs.DB.First(&chunk, "file_id = ? AND chunk_index = ?", file.ID, chunkIndex).Error; err != nil {
		return err
	}

	chunk.IsAvailable = true
	return cs.DB.Save(&chunk).Error
}
