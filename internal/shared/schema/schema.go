package schema

type File struct {
	ID           uint `gorm:"primaryKey"`
	Name         string
	Size         int
	MaxChunkSize int
	TotalChunks  int
	Checksum     string
	CreatedAt    int64
}

type Chunk struct {
	ID       uint `gorm:"primaryKey"`
	Index    int
	FileID   uint `gorm:"not null;foreignKey:FileID;constraint:OnDelete:CASCADE"`
	File     File `gorm:"constraint:OnDelete:CASCADE"`
	Size     int
	Checksum string
}

type Peer struct {
	ID        uint `gorm:"primaryKey"`
	IPAddress string
	Port      string
}

type PeerChunk struct {
	ID      uint  `gorm:"primaryKey"`
	PeerID  uint  `gorm:"not null;foreignKey:PeerID;constraint:OnDelete:CASCADE"`
	Peer    Peer  `gorm:"constraint:OnDelete:CASCADE"`
	ChunkID uint  `gorm:"not null;foreignKey:ChunkID;constraint:OnDelete:CASCADE"`
	Chunk   Chunk `gorm:"constraint:OnDelete:CASCADE"`
}
