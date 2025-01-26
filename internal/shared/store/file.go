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

func (fs *FileStore) CreateFile(file *schema.File) error {
	return fs.DB.Create(&file).Error
}
