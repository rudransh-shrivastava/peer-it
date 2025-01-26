package store

import (
	"github.com/rudransh-shrivastava/peer-it/internal/shared/schema"
	"gorm.io/gorm"
)

type ClientStore struct {
	DB *gorm.DB
}

func NewClientStore(db *gorm.DB) *ClientStore {
	return &ClientStore{DB: db}
}

func (cs *ClientStore) CreateClient(ip string, port string) error {
	client := schema.Peer{IPAddress: ip, Port: port}
	return cs.DB.Create(&client).Error
}

func (cs *ClientStore) GetClients() ([]schema.Peer, error) {
	clients := []schema.Peer{}
	err := cs.DB.Find(&clients).Error
	if err != nil {
		return nil, err
	}
	return clients, nil
}

func (cs *ClientStore) DeleteClient(ip string, port string) error {
	return cs.DB.Where("ip_address = ? AND port = ?", ip, port).Delete(&schema.Peer{}).Error
}
