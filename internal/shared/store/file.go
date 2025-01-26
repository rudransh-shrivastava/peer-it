package store

import (
	"fmt"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/schema"
	"gorm.io/gorm"
)

type FileStore struct {
	DB *gorm.DB
}

func NewFileStore(db *gorm.DB) *FileStore {
	return &FileStore{DB: db}
}

func (fs *FileStore) CreateFile(file *schema.File) {
	fmt.Println("creating the file ----++++=----")
	fs.DB.Create(&file)
}
