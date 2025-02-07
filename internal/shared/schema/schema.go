package schema

// Requires file store
type File struct {
	ID           uint `gorm:"primaryKey"`
	Name         string
	Size         int64
	MaxChunkSize int
	TotalChunks  int
	Hash         string
	CreatedAt    int64
}

// Requires chunk store
type Chunk struct {
	ID          uint `gorm:"primaryKey"`
	FileID      uint `gorm:"not null;foreignKey:FileID;constraint:OnDelete:CASCADE"`
	File        File `gorm:"constraint:OnDelete:CASCADE"`
	ChunkIndex  int
	ChunkSize   int
	ChunkHash   string
	IsAvailable bool
}

// Requires peer store
type Peer struct {
	ID        uint `gorm:"primaryKey"`
	IPAddress string
	Port      string
}

// Relation between peers and files
type Swarm struct {
	ID     uint `gorm:"primaryKey"`
	PeerID uint `gorm:"not null;foreignKey:PeerID;constraint:OnDelete:CASCADE"`
	Peer   Peer `gorm:"constraint:OnDelete:CASCADE"`
	FileID uint `gorm:"not null;foreignKey:FileID;constraint:OnDelete:CASCADE"`
	File   File `gorm:"constraint:OnDelete:CASCADE"`
}
