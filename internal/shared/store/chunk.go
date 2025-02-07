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
		File:        *(file),
		ChunkIndex:  index,
		ChunkSize:   size,
		ChunkHash:   hash,
		IsAvailable: isAvailable,
	}
	err := cs.DB.Create(&chunk).Error
	if err != nil {
		return err
	}
	return nil
}

func (cs *ChunkStore) GetChunk(fileHash string, chunkIndex int) (*schema.Chunk, error) {
	file := &schema.File{}
	err := cs.DB.First(&file, "hash = ?", fileHash).Error
	chunk := schema.Chunk{}
	err = cs.DB.First(&chunk, "file_id = ? AND chunk_index = ?", file.ID, chunkIndex).Error
	if err != nil {
		return nil, err
	}

	return &chunk, nil
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

func (cs *ChunkStore) MarkChunkAvailable(filehash string, chunkIndex int) error {
	file := &schema.File{}
	err := cs.DB.First(&file, "hash = ?", filehash).Error
	chunk := schema.Chunk{}
	err = cs.DB.First(&chunk, "file_id = ? AND chunk_index = ?", file.ID, chunkIndex).Error
	if err != nil {
		return err
	}
	chunk.IsAvailable = true
	return cs.DB.Save(&chunk).Error
}
