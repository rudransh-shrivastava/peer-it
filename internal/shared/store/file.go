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
	_, err := fs.GetFileByChecksum(file.Checksum)
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

func (fs *FileStore) GetFileByChecksum(checksum string) (*schema.File, error) {
	file := &schema.File{}
	if err := fs.DB.First(file, "checksum = ?", checksum).Error; err != nil {
		return nil, err
	}
	return file, nil
}
