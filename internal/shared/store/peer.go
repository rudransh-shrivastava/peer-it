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

func (ps *PeerStore) CreatePeer(ip string, port string) error {
	peer := schema.Peer{IPAddress: ip, Port: port}
	return ps.DB.Create(&peer).Error
}

func (ps *PeerStore) DeletePeer(ip string, port string) error {
	return ps.DB.Where("ip_address = ? AND port = ?", ip, port).Delete(&schema.Peer{}).Error
}

func (ps *PeerStore) AddPeerToSwarm(ip string, port string, fileHash string) error {
	file := &schema.File{}
	err := ps.DB.First(&file, "hash = ?", fileHash).Error
	if err != nil {
		return err
	}

	peer := &schema.Peer{}
	err = ps.DB.First(&peer, "ip_address = ? AND port = ?", ip, port).Error
	if err != nil {
		return err
	}

	swarm := &schema.Swarm{
		File: *file,
		Peer: *peer,
	}
	return ps.DB.Create(&swarm).Error
}

// TODO: good error handling
func (ps *PeerStore) DropAllPeers() error {
	err := ps.DB.Exec("DELETE FROM swarms").Error
	if err != nil {
		return err
	}
	err = ps.DB.Exec("DELETE FROM peer_listeners").Error
	if err != nil {
		return err
	}
	return ps.DB.Exec("DELETE FROM peers").Error
}

func (ps *PeerStore) GetPeersByFileHash(fileHash string) ([]schema.Peer, error) {
	peers := []schema.Peer{}
	err := ps.DB.Raw("SELECT * FROM peers WHERE id IN (SELECT peer_id FROM swarms WHERE file_id IN (SELECT id FROM files WHERE hash = ?))", fileHash).Scan(&peers).Error
	if err != nil {
		return nil, err
	}
	return peers, nil
}
