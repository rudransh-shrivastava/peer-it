package node

import (
	"context"
	"fmt"

	"github.com/pion/webrtc/v3"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"google.golang.org/protobuf/proto"
)

func (n *Node) handleWebRTCConnection(peerId string, fileHash string, config webrtc.Configuration, isInitiator bool) error {
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return fmt.Errorf("failed to create peer connection: %v", err)
	}
	n.mu.Lock()
	n.peerConnections[peerId] = peerConnection
	n.mu.Unlock()
	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		n.logger.Warnf("Peer Connection State has changed: %s", s.String())
		if s == webrtc.PeerConnectionStateDisconnected {
			n.logger.Infof("Peer %s disconnected", peerId)
			n.mu.Lock()
			delete(n.peerConnections, peerId)
			n.mu.Unlock()
			n.removePeerFromDownloads(peerId)
		}
	})

	setupDataChannel := func(dc *webrtc.DataChannel) {
		n.mu.Lock()
		n.peerDataChannels[peerId] = dc
		n.mu.Unlock()

		dc.OnOpen(func() {
			n.logger.Debugf("Data channel '%s'-'%d' open", dc.Label(), dc.ID())
			if isInitiator {
				n.logger.Infof("Sending chunk maps for file: %s", fileHash)
				file, err := n.files.GetFileByHash(context.Background(), fileHash)
				if err != nil {
					n.logger.Errorf("Failed to get file: %v", err)
					return
				}

				if err := n.sendIntroduction(peerId, file.Hash); err != nil {
					n.logger.Warnf("Failed to send introduction: %v", err)
				}
			}
		})

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			n.webrtcMessageHandler(msg, peerId)
		})

		dc.OnError(func(err error) {
			n.logger.Errorf("Data channel error: %v", err)
		})

		dc.OnClose(func() {
			n.logger.Debugf("Data channel '%s'-'%d' closed", dc.Label(), dc.ID())
			n.mu.Lock()
			delete(n.peerDataChannels, peerId)
			n.mu.Unlock()
			// Add peer to our peer list
			n.logger.Infof("Removing peer %s from our swarm", peerId)
			n.removePeerFromDownloads(peerId)
		})
	}

	if isInitiator {
		// We create the data channel
		n.logger.Debug("Creating data channel as we are the initiator")
		protocolName := "file-transfer"
		dataChannelConfig := &webrtc.DataChannelInit{
			Ordered:        &[]bool{true}[0], // Ensure ordered delivery
			MaxRetransmits: nil,              // Unlimited retransmissions
			Protocol:       &protocolName,
		}

		dataChannel, err := peerConnection.CreateDataChannel("data", dataChannelConfig)
		if err != nil {
			return fmt.Errorf("failed to create data channel: %v", err)
		}
		setupDataChannel(dataChannel)
	} else {
		// We wait for the data channel
		n.logger.Info("Waiting for data channel as other peer is the initiator")
		peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
			setupDataChannel(dc)
		})
	}

	peerConnection.OnICECandidate(func(ice *webrtc.ICECandidate) {
		if ice != nil {
			err := n.trackerRouter.WriteMessage(BuildICECandidateMessage(peerId, ice))
			if err != nil {
				n.logger.Warnf("Failed to send ICE candidate: %v", err)
			}
		}
	})
	return nil
}

func (n *Node) webrtcMessageHandler(msg webrtc.DataChannelMessage, peerId string) {
	var netMsg protocol.NetworkMessage
	if err := proto.Unmarshal(msg.Data, &netMsg); err != nil {
		n.logger.Warnf("Failed to unmarshal message: %v", err)
		return
	}

	switch m := netMsg.MessageType.(type) {
	case *protocol.NetworkMessage_Introduction:
		n.logger.Infof("Received introduction message for file: %s", m.Introduction.FileHash)
		n.handleIntroduction(peerId, m.Introduction)

	case *protocol.NetworkMessage_ChunkRequest:
		n.logger.Infof("Received chunk request for file: %s", m.ChunkRequest.FileHash)
		n.handleChunkRequest(peerId, m.ChunkRequest)

	case *protocol.NetworkMessage_ChunkResponse:
		n.logger.Infof("Received chunk response for file: %s", m.ChunkResponse.FileHash)
		n.handleChunkResponse(peerId, m.ChunkResponse)

	default:
		n.logger.Warnf("Unknown message type received")
	}
}
