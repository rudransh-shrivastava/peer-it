package schema

// Requires file store
type File struct {
	ID           uint `gorm:"primaryKey"`
	Size         int64
	MaxChunkSize int
	TotalChunks  int
	Hash         string
	CreatedAt    int64
}

// Requires chunk store
type Chunk struct {
	ID     uint `gorm:"primaryKey"`
	Index  int
	FileID uint `gorm:"not null;foreignKey:FileID;constraint:OnDelete:CASCADE"`
	File   File `gorm:"constraint:OnDelete:CASCADE"`
}

type ChunkMetadata struct {
	ID        uint `gorm:"primaryKey"`
	ChunkID   uint `gorm:"not null;foreignKey:ChunkID;constraint:OnDelete:CASCADE"`
	Chunk     Chunk
	ChunkSize int
	ChunkHash string
}

// Requires peer store
type Peer struct {
	ID        uint `gorm:"primaryKey"`
	IPAddress string
	Port      string
}

type PeerListener struct {
	ID               uint `gorm:"primaryKey"`
	PeerConnID       string
	PeerID           uint `gorm:"not null;foreignKey:PeerID;constraint:OnDelete:CASCADE"`
	Peer             Peer
	PublicListenPort string
	PublicIpAddress  string
}

// Relation between peers and files
type Swarm struct {
	ID     uint `gorm:"primaryKey"`
	PeerID uint `gorm:"not null;foreignKey:PeerID;constraint:OnDelete:CASCADE"`
	Peer   Peer `gorm:"constraint:OnDelete:CASCADE"`
	FileID uint `gorm:"not null;foreignKey:FileID;constraint:OnDelete:CASCADE"`
	File   File `gorm:"constraint:OnDelete:CASCADE"`
}
