package protocol

const (
	MaxPayloadSize = 1280

	MaxChunkSize = 1024 * 1024

	HashSize = 32

	NodeIDSize = 16
)

type MessageType uint16

const (
	MsgPing MessageType = 0x0001
	MsgPong MessageType = 0x0002

	MsgFileListReq MessageType = 0x0010
	MsgFileListRes MessageType = 0x0011

	MsgFileMetaReq MessageType = 0x0020
	MsgFileMetaRes MessageType = 0x0021

	MsgChunkReq MessageType = 0x0030
	MsgChunkRes MessageType = 0x0031

	MsgPeerAnnounce MessageType = 0x0040
	MsgPeerListReq  MessageType = 0x0041
	MsgPeerListRes  MessageType = 0x0042

	MsgDiscovery MessageType = 0x0050

	MsgError MessageType = 0x00FF
)

func (t MessageType) String() string {
	switch t {
	case MsgPing:
		return "PING"
	case MsgPong:
		return "PONG"
	case MsgFileListReq:
		return "FILE_LIST_REQ"
	case MsgFileListRes:
		return "FILE_LIST_RES"
	case MsgFileMetaReq:
		return "FILE_META_REQ"
	case MsgFileMetaRes:
		return "FILE_META_RES"
	case MsgChunkReq:
		return "CHUNK_REQ"
	case MsgChunkRes:
		return "CHUNK_RES"
	case MsgPeerAnnounce:
		return "PEER_ANNOUNCE"
	case MsgPeerListReq:
		return "PEER_LIST_REQ"
	case MsgPeerListRes:
		return "PEER_LIST_RES"
	case MsgDiscovery:
		return "DISCOVERY"
	case MsgError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

type ErrorCode uint16

const (
	ErrUnknown       ErrorCode = 0x0000
	ErrInvalidMsg    ErrorCode = 0x0001
	ErrFileNotFound  ErrorCode = 0x0002
	ErrChunkNotFound ErrorCode = 0x0003
	ErrPeerNotFound  ErrorCode = 0x0004
	ErrInternal      ErrorCode = 0x00FF
)

func (e ErrorCode) String() string {
	switch e {
	case ErrUnknown:
		return "UNKNOWN"
	case ErrInvalidMsg:
		return "INVALID_MESSAGE"
	case ErrFileNotFound:
		return "FILE_NOT_FOUND"
	case ErrChunkNotFound:
		return "CHUNK_NOT_FOUND"
	case ErrPeerNotFound:
		return "PEER_NOT_FOUND"
	case ErrInternal:
		return "INTERNAL_ERROR"
	default:
		return "UNKNOWN"
	}
}
