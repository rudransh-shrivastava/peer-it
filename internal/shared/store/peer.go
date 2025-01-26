package store

import (
	"github.com/rudransh-shrivastava/peer-it/internal/shared/schema"
	"gorm.io/gorm"
)

type PeerStore struct {
	DB *gorm.DB
}

func NewPeerStore(db *gorm.DB) *PeerStore {
	return &PeerStore{DB: db}
}

func (cs *PeerStore) CreatePeer(ip string, port string) error {
	peer := schema.Peer{IPAddress: ip, Port: port}
	return cs.DB.Create(&peer).Error
}

func (cs *PeerStore) DeletePeer(ip string, port string) error {
	return cs.DB.Where("ip_address = ? AND port = ?", ip, port).Delete(&schema.Peer{}).Error
}
