package node

import (
	"github.com/pion/webrtc/v3"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
)

func BuildICECandidateMessage(targetPeerID string, ice *webrtc.ICECandidate) *protocol.NetworkMessage {
	json := ice.ToJSON()
	return &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Signaling{
			Signaling: &protocol.SignalingMessage{
				TargetPeerId: targetPeerID,
				Message: &protocol.SignalingMessage_IceCandidate{
					IceCandidate: &protocol.IceCandidate{
						Candidate:     json.Candidate,
						SdpMid:        *json.SDPMid,
						SdpMlineIndex: uint32(*json.SDPMLineIndex),
					},
				},
			},
		},
	}
}

func BuildSignalingOfferMessage(targetPeerID, sdp string) *protocol.NetworkMessage {
	return &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Signaling{
			Signaling: &protocol.SignalingMessage{
				TargetPeerId: targetPeerID,
				Message: &protocol.SignalingMessage_Offer{
					Offer: &protocol.Offer{Sdp: sdp},
				},
			},
		},
	}
}

func BuildSignalingAnswerMessage(targetPeerID, sdp string) *protocol.NetworkMessage {
	return &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Signaling{
			Signaling: &protocol.SignalingMessage{
				TargetPeerId: targetPeerID,
				Message: &protocol.SignalingMessage_Answer{
					Answer: &protocol.Answer{Sdp: sdp},
				},
			},
		},
	}
}

func BuildAnnounceMessage(files []*protocol.FileInfo) *protocol.NetworkMessage {
	return &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Announce{
			Announce: &protocol.AnnounceMessage{Files: files},
		},
	}
}

func BuildPeerListRequestMessage(fileHash string) *protocol.NetworkMessage {
	return &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_PeerListRequest{
			PeerListRequest: &protocol.PeerListRequest{FileHash: fileHash},
		},
	}
}

func BuildHeartbeatMessage(timestamp int64) *protocol.NetworkMessage {
	return &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Heartbeat{
			Heartbeat: &protocol.HeartbeatMessage{Timestamp: timestamp},
		},
	}
}

func BuildLogMessage(msg string) *protocol.NetworkMessage {
	return &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Log{
			Log: &protocol.LogMessage{Message: msg},
		},
	}
}

func BuildIntroductionMessage(fileHash string, chunksMap []int32) *protocol.NetworkMessage {
	return &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_Introduction{
			Introduction: &protocol.IntroductionMessage{
				FileHash:  fileHash,
				ChunksMap: chunksMap,
			},
		},
	}
}

func BuildChunkRequestMessage(fileHash string, chunkIndex int32) *protocol.NetworkMessage {
	return &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_ChunkRequest{
			ChunkRequest: &protocol.ChunkRequest{
				FileHash:   fileHash,
				ChunkIndex: chunkIndex,
			},
		},
	}
}

func BuildChunkResponseMessage(fileHash string, chunkIndex int32, data []byte) *protocol.NetworkMessage {
	return &protocol.NetworkMessage{
		MessageType: &protocol.NetworkMessage_ChunkResponse{
			ChunkResponse: &protocol.ChunkResponse{
				FileHash:   fileHash,
				ChunkIndex: chunkIndex,
				ChunkData:  data,
			},
		},
	}
}
