package protocol

type Message interface {
	Type() MessageType
}

type Ping struct{}

func (Ping) Type() MessageType { return MsgPing }

type Pong struct{}

func (Pong) Type() MessageType { return MsgPong }

type FileListReq struct{}

func (FileListReq) Type() MessageType { return MsgFileListReq }

type FileEntry struct {
	Hash [HashSize]byte
	Size uint64 // File size in bytes
	Name string
}

type FileListRes struct {
	Files []FileEntry
}

func (FileListRes) Type() MessageType { return MsgFileListRes }

type FileMetaReq struct {
	Hash [HashSize]byte
}

func (FileMetaReq) Type() MessageType { return MsgFileMetaReq }

type ChunkMeta struct {
	Index uint32
	Size  uint32
	Hash  [HashSize]byte
}

type FileMetaRes struct {
	Hash         [HashSize]byte
	Size         uint64
	Name         string
	MaxChunkSize uint32
	Chunks       []ChunkMeta
}

func (FileMetaRes) Type() MessageType { return MsgFileMetaRes }

type ChunkReq struct {
	FileHash   [HashSize]byte
	ChunkIndex uint32
}

func (ChunkReq) Type() MessageType { return MsgChunkReq }

type ChunkRes struct {
	FileHash   [HashSize]byte
	ChunkIndex uint32
	Data       []byte
}

func (ChunkRes) Type() MessageType { return MsgChunkRes }

type PeerAnnounce struct {
	NodeID    [NodeIDSize]byte
	Port      uint16
	FileCount uint16
}

func (PeerAnnounce) Type() MessageType { return MsgPeerAnnounce }

type PeerListReq struct {
	FileHash [HashSize]byte
}

func (PeerListReq) Type() MessageType { return MsgPeerListReq }

type PeerInfo struct {
	NodeID [NodeIDSize]byte
	IP     [16]byte
	Port   uint16
}

type PeerListRes struct {
	FileHash [HashSize]byte
	Peers    []PeerInfo
}

func (PeerListRes) Type() MessageType { return MsgPeerListRes }

type HolePunchReq struct {
	TargetNodeID [NodeIDSize]byte
	TargetIP     [16]byte
	TargetPort   uint16
}

func (HolePunchReq) Type() MessageType { return MsgHolePunchReq }

type HolePunchProbe struct {
	SenderNodeID [NodeIDSize]byte
}

func (HolePunchProbe) Type() MessageType { return MsgHolePunchProbe }

type Discovery struct {
	NodeID    [NodeIDSize]byte
	Port      uint16
	FileCount uint16
}

func (Discovery) Type() MessageType { return MsgDiscovery }

type Error struct {
	Code    ErrorCode
	Message string
}

func (Error) Type() MessageType { return MsgError }
