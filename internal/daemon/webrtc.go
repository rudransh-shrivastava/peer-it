package daemon

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pion/webrtc/v3"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"google.golang.org/protobuf/proto"
)

func (d *Daemon) handleWebRTCConnection(peerId string, fileHash string, config webrtc.Configuration, isInitiator bool) error {
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return fmt.Errorf("failed to create peer connection: %v", err)
	}
	d.mu.Lock()
	d.PeerConnections[peerId] = peerConnection
	d.mu.Unlock()
	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		d.Logger.Warnf("Peer Connection State has changed: %s", s.String())
		if s == webrtc.PeerConnectionStateDisconnected {
			d.Logger.Infof("Peer %s disconnected", peerId)
			d.mu.Lock()
			delete(d.PeerConnections, peerId)
			d.mu.Unlock()
			d.removePeerFromDownloads(peerId)
		}
	})

	setupDataChannel := func(dc *webrtc.DataChannel) {
		d.mu.Lock()
		d.PeerDataChannels[peerId] = dc
		d.mu.Unlock()

		dc.OnOpen(func() {
			d.Logger.Debugf("Data channel '%s'-'%d' open", dc.Label(), dc.ID())
			if isInitiator {
				d.Logger.Infof("Sending chunk maps for file: %s", fileHash)
				file, err := d.FileStore.GetFileByHash(context.Background(), fileHash)
				if err != nil {
					d.Logger.Errorf("Failed to get file: %v", err)
					return
				}

				if err := d.sendIntroduction(peerId, file.Hash); err != nil {
					d.Logger.Warnf("Failed to send introduction: %v", err)
				}
			}
		})

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			d.webrtcMessageHandler(msg, peerId)
		})

		dc.OnError(func(err error) {
			d.Logger.Errorf("Data channel error: %v", err)
		})

		dc.OnClose(func() {
			d.Logger.Debugf("Data channel '%s'-'%d' closed", dc.Label(), dc.ID())
			d.mu.Lock()
			delete(d.PeerDataChannels, peerId)
			d.mu.Unlock()
			// Add peer to our peer list
			d.Logger.Infof("Removing peer %s from our swarm", peerId)
			d.removePeerFromDownloads(peerId)
		})
	}

	if isInitiator {
		// We create the data channel
		d.Logger.Debug("Creating data channel as we are the initiator")
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
		d.Logger.Info("Waiting for data channel as other peer is the initiator")
		peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
			setupDataChannel(dc)
		})
	}

	peerConnection.OnICECandidate(func(ice *webrtc.ICECandidate) {
		if ice != nil {
			_, err := json.Marshal(ice.ToJSON())
			if err != nil {
				d.Logger.Warnf("Failed to marshal ICE candidate: %v", err)
				return
			}

			signalingMsg := &protocol.NetworkMessage{
				MessageType: &protocol.NetworkMessage_Signaling{
					Signaling: &protocol.SignalingMessage{
						TargetPeerId: peerId,
						Message: &protocol.SignalingMessage_IceCandidate{
							IceCandidate: &protocol.IceCandidate{
								Candidate:     ice.ToJSON().Candidate,
								SdpMid:        *ice.ToJSON().SDPMid,
								SdpMlineIndex: uint32(*ice.ToJSON().SDPMLineIndex),
							},
						},
					},
				},
			}

			err = d.TrackerRouter.WriteMessage(signalingMsg)
			if err != nil {
				d.Logger.Warnf("Failed to send ICE candidate: %v", err)
			}
		}
	})
	return nil
}

func (d *Daemon) webrtcMessageHandler(msg webrtc.DataChannelMessage, peerId string) {
	var netMsg protocol.NetworkMessage
	if err := proto.Unmarshal(msg.Data, &netMsg); err != nil {
		d.Logger.Warnf("Failed to unmarshal message: %v", err)
		return
	}

	switch m := netMsg.MessageType.(type) {
	case *protocol.NetworkMessage_Introduction:
		d.Logger.Infof("Received introduction message for file: %s", m.Introduction.FileHash)
		d.handleIntroduction(peerId, m.Introduction)

	case *protocol.NetworkMessage_ChunkRequest:
		d.Logger.Infof("Received chunk request for file: %s", m.ChunkRequest.FileHash)
		d.handleChunkRequest(peerId, m.ChunkRequest)

	case *protocol.NetworkMessage_ChunkResponse:
		d.Logger.Infof("Received chunk response for file: %s", m.ChunkResponse.FileHash)
		d.handleChunkResponse(peerId, m.ChunkResponse)

	// TODO: case for a GOT message to tell others we got a chunk
	default:
		d.Logger.Warnf("Unknown message type received")
	}
}
