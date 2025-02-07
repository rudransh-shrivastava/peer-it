package store

import (
	"github.com/rudransh-shrivastava/peer-it/internal/shared/schema"
	"gorm.io/gorm"
)

type FileStore struct {
	DB *gorm.DB
}

func NewFileStore(db *gorm.DB) *FileStore {
	return &FileStore{DB: db}
}

func (fs *FileStore) CreateFile(file *schema.File) (bool, error) {
	_, err := fs.GetFileByHash(file.Hash)
	if err != nil {
		// create the new record
		if err := fs.DB.Create(file).Error; err != nil {
			return false, err // db error
		}
		return true, nil
	}
	return false, nil // already exists
}

func (fs *FileStore) GetFiles() ([]schema.File, error) {
	files := []schema.File{}
	err := fs.DB.Find(&files).Error
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (fs *FileStore) GetChunks(fileHash string) ([]schema.Chunk, error) {
	chunks := []schema.Chunk{}
	err := fs.DB.Raw("SELECT * FROM chunks WHERE file_id IN (SELECT id FROM files WHERE hash = ?)", fileHash).Scan(&chunks).Error
	if err != nil {
		return nil, err
	}
	return chunks, nil
}

func (fs *FileStore) GetFileByHash(hash string) (*schema.File, error) {
	file := &schema.File{}
	if err := fs.DB.First(file, "hash = ?", hash).Error; err != nil {
		return nil, err
	}
	return file, nil
}

func (fs *FileStore) GetFileNameByHash(hash string) (string, error) {
	file := &schema.File{}
	if err := fs.DB.First(file, "hash = ?", hash).Error; err != nil {
		return "", err
	}
	return file.Name, nil
}
