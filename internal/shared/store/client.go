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

func (cs *ClientStore) CreateClient(ip string, port string) {
	client := schema.Peer{IPAddress: ip, Port: port}
	cs.DB.Create(&client)
}

func (cs *ClientStore) GetClients() []schema.Peer {
	clients := []schema.Peer{}
	cs.DB.Find(&clients)
	return clients
}

func (cs *ClientStore) DeleteClient(ip string) {
	cs.DB.Where("ip_address = ?", ip).Delete(&schema.Peer{})
}
