package protocol

type Message interface {
	Type() MessageType
}

type ChunkMeta struct {
	Hash  [HashSize]byte
	Index uint32
	Size  uint32
}

type ChunkReq struct {
	ChunkIndex uint32
	FileHash   [HashSize]byte
}

func (ChunkReq) Type() MessageType { return MsgChunkReq }

type ChunkRes struct {
	ChunkIndex uint32
	Data       []byte
	FileHash   [HashSize]byte
}

func (ChunkRes) Type() MessageType { return MsgChunkRes }

type Discovery struct {
	FileCount uint16
	NodeID    [NodeIDSize]byte
	Port      uint16
}

func (Discovery) Type() MessageType { return MsgDiscovery }

type Error struct {
	Code    ErrorCode
	Message string
}

func (Error) Type() MessageType { return MsgError }

type FileEntry struct {
	Hash [HashSize]byte
	Name string
	Size uint64
}

type FileListReq struct{}

func (FileListReq) Type() MessageType { return MsgFileListReq }

type FileListRes struct {
	Files []FileEntry
}

func (FileListRes) Type() MessageType { return MsgFileListRes }

type FileMetaReq struct {
	Hash [HashSize]byte
}

func (FileMetaReq) Type() MessageType { return MsgFileMetaReq }

type FileMetaRes struct {
	Chunks       []ChunkMeta
	Hash         [HashSize]byte
	MaxChunkSize uint32
	Name         string
	Size         uint64
}

func (FileMetaRes) Type() MessageType { return MsgFileMetaRes }

type HolePunchProbe struct {
	SenderNodeID [NodeIDSize]byte
}

func (HolePunchProbe) Type() MessageType { return MsgHolePunchProbe }

type HolePunchReq struct {
	TargetIP     [16]byte
	TargetNodeID [NodeIDSize]byte
	TargetPort   uint16
}

func (HolePunchReq) Type() MessageType { return MsgHolePunchReq }

type PeerAnnounce struct {
	FileCount uint16
	NodeID    [NodeIDSize]byte
	Port      uint16
}

func (PeerAnnounce) Type() MessageType { return MsgPeerAnnounce }

type PeerInfo struct {
	IP     [16]byte
	NodeID [NodeIDSize]byte
	Port   uint16
}

type PeerListReq struct {
	FileHash [HashSize]byte
}

func (PeerListReq) Type() MessageType { return MsgPeerListReq }

type PeerListRes struct {
	FileHash [HashSize]byte
	Peers    []PeerInfo
}

func (PeerListRes) Type() MessageType { return MsgPeerListRes }

type Ping struct{}

func (Ping) Type() MessageType { return MsgPing }

type Pong struct{}

func (Pong) Type() MessageType { return MsgPong }
