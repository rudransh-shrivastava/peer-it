package protocol

type Message interface {
	Type() MessageType
}

type NodeID [NodeIDSize]byte

type CallReq struct {
	TargetNodeID NodeID
}

func (CallReq) Type() MessageType { return MsgCallReq }

type ChunkMeta struct {
	Hash  FileHash
	Index uint32
	Size  uint32
}

type ChunkReq struct {
	ChunkIndex uint32
	FileHash   FileHash
}

func (ChunkReq) Type() MessageType { return MsgChunkReq }

type ChunkRes struct {
	ChunkIndex uint32
	Data       []byte
	FileHash   FileHash
}

func (ChunkRes) Type() MessageType { return MsgChunkRes }

type Discovery struct {
	NodeID NodeID
	Port   uint16
}

func (Discovery) Type() MessageType { return MsgDiscovery }

type Error struct {
	Code    ErrorCode
	Message string
}

func (Error) Type() MessageType { return MsgError }

type FileEntry struct {
	Hash FileHash
	Name string
	Size uint64
}

type FileHash [HashSize]byte

type FileListReq struct{}

func (FileListReq) Type() MessageType { return MsgFileListReq }

// TODO(rudransh-shrivastava): Extend with peer count for each file.
type FileListRes struct {
	Files []FileEntry
}

func (FileListRes) Type() MessageType { return MsgFileListRes }

type FileMetaReq struct {
	Hash FileHash
}

func (FileMetaReq) Type() MessageType { return MsgFileMetaReq }

type FileMetaRes struct {
	Chunks       []ChunkMeta
	Hash         FileHash
	MaxChunkSize uint32
	Name         string
	Size         uint64
}

func (FileMetaRes) Type() MessageType { return MsgFileMetaRes }

type HolePunchProbe struct {
	NodeID NodeID
	TxnID  [TxnIDSize]byte
}

func (HolePunchProbe) Type() MessageType { return MsgHolePunchProbe }

type PeerAnnounce struct {
	FileCount uint16
	Files     []FileEntry
}

func (PeerAnnounce) Type() MessageType { return MsgPeerAnnounce }

type PeerInfo struct {
	NodeID NodeID
}

type PeerListReq struct {
	FileHash FileHash
}

func (PeerListReq) Type() MessageType { return MsgPeerListReq }

type PeerListRes struct {
	FileHash FileHash
	Peers    []PeerInfo
}

func (PeerListRes) Type() MessageType { return MsgPeerListRes }

type Ping struct{}

func (Ping) Type() MessageType { return MsgPing }

type Pong struct{}

func (Pong) Type() MessageType { return MsgPong }
