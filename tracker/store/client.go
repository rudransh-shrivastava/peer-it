package store

import (
	"github.com/rudransh-shrivastava/peer-it/tracker/db"
	"gorm.io/gorm"
)

type ClientStore struct {
	DB *gorm.DB
}

func NewClientStore(db *gorm.DB) *ClientStore {
	return &ClientStore{DB: db}
}

func (cs *ClientStore) CreateClient(ip string, port string) {
	client := db.Client{IPAddress: ip, Port: port}
	cs.DB.Create(&client)
}

func (cs *ClientStore) GetClients() []db.Client {
	clients := []db.Client{}
	cs.DB.Find(&clients)
	return clients
}

func (cs *ClientStore) DeleteClient(ip string) {
	cs.DB.Where("ip_address = ?", ip).Delete(&db.Client{})
}
