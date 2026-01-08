package protocol

const (
	HashSize       = 32
	MaxChunkSize   = 1024 * 1024
	MaxPayloadSize = 1280
	NodeIDSize     = 16
)

type MessageType uint16

const (
	MsgChunkReq       MessageType = 0x0030
	MsgChunkRes       MessageType = 0x0031
	MsgDiscovery      MessageType = 0x0050
	MsgError          MessageType = 0x00FF
	MsgFileListReq    MessageType = 0x0010
	MsgFileListRes    MessageType = 0x0011
	MsgFileMetaReq    MessageType = 0x0020
	MsgFileMetaRes    MessageType = 0x0021
	MsgHolePunchProbe MessageType = 0x0061
	MsgHolePunchReq   MessageType = 0x0060
	MsgPeerAnnounce   MessageType = 0x0040
	MsgPeerListReq    MessageType = 0x0041
	MsgPeerListRes    MessageType = 0x0042
	MsgPing           MessageType = 0x0001
	MsgPong           MessageType = 0x0002
)

func (t MessageType) String() string {
	switch t {
	case MsgChunkReq:
		return "CHUNK_REQ"
	case MsgChunkRes:
		return "CHUNK_RES"
	case MsgDiscovery:
		return "DISCOVERY"
	case MsgError:
		return "ERROR"
	case MsgFileListReq:
		return "FILE_LIST_REQ"
	case MsgFileListRes:
		return "FILE_LIST_RES"
	case MsgFileMetaReq:
		return "FILE_META_REQ"
	case MsgFileMetaRes:
		return "FILE_META_RES"
	case MsgHolePunchProbe:
		return "HOLE_PUNCH_PROBE"
	case MsgHolePunchReq:
		return "HOLE_PUNCH_REQ"
	case MsgPeerAnnounce:
		return "PEER_ANNOUNCE"
	case MsgPeerListReq:
		return "PEER_LIST_REQ"
	case MsgPeerListRes:
		return "PEER_LIST_RES"
	case MsgPing:
		return "PING"
	case MsgPong:
		return "PONG"
	default:
		return "UNKNOWN"
	}
}

type ErrorCode uint16

const (
	ErrChunkNotFound ErrorCode = 0x0003
	ErrFileNotFound  ErrorCode = 0x0002
	ErrInternal      ErrorCode = 0x00FF
	ErrInvalidMsg    ErrorCode = 0x0001
	ErrPeerNotFound  ErrorCode = 0x0004
	ErrUnknown       ErrorCode = 0x0000
)

func (e ErrorCode) String() string {
	switch e {
	case ErrChunkNotFound:
		return "CHUNK_NOT_FOUND"
	case ErrFileNotFound:
		return "FILE_NOT_FOUND"
	case ErrInternal:
		return "INTERNAL_ERROR"
	case ErrInvalidMsg:
		return "INVALID_MESSAGE"
	case ErrPeerNotFound:
		return "PEER_NOT_FOUND"
	case ErrUnknown:
		return "UNKNOWN"
	default:
		return "UNKNOWN"
	}
}
