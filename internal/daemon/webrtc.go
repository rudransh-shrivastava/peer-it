package daemon

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/pion/webrtc/v3"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"google.golang.org/protobuf/proto"
)

func (d *Daemon) handleWebRTCConnection(peerId string, config webrtc.Configuration) error {
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return fmt.Errorf("failed to create peer connection: %v", err)
	}

	d.PeerConnections[peerId] = peerConnection

	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		d.Logger.Infof("Peer Connection State has changed: %s", s.String())
	})

	setupDataChannel := func(dc *webrtc.DataChannel) {
		d.mu.Lock()
		d.PeerDataChannels[peerId] = dc
		d.mu.Unlock()

		dc.OnOpen(func() {
			d.Logger.Infof("Data channel '%s'-'%d' open", dc.Label(), dc.ID())
			d.Logger.Infof("Sending chunk maps for all files")

			files, err := d.FileStore.GetFiles()
			if err != nil {
				d.Logger.Errorf("Failed to get files: %v", err)
				return
			}
			d.Logger.Infof("Found %d files in FileStore", len(files))

			for _, file := range files {
				if err := d.sendIntroduction(peerId, file.Hash); err != nil {
					d.Logger.Warnf("Failed to send introduction: %v", err)
				}
			}
		})

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			d.Logger.Infof("Received message from DataChannel '%s' with length: %d", dc.Label(), len(msg.Data))
			d.webrtcMessageHandler(msg, peerId)
		})

		dc.OnError(func(err error) {
			d.Logger.Errorf("Data channel error: %v", err)
		})

		dc.OnClose(func() {
			d.Logger.Infof("Data channel '%s'-'%d' closed", dc.Label(), dc.ID())
		})
	}

	myID, err := strconv.Atoi(d.ID)
	if err != nil {
		return fmt.Errorf("invalid local peer ID format: %v", err)
	}

	otherID, err := strconv.Atoi(peerId)
	if err != nil {
		return fmt.Errorf("invalid remote peer ID format: %v", err)
	}

	if myID < otherID {
		// We create the data channel
		d.Logger.Infof("Creating data channel as lower peer ID (%d < %d)", myID, otherID)
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
		d.Logger.Infof("Waiting for data channel as higher peer ID (%d > %d)", myID, otherID)
		peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
			d.Logger.Infof("Received data channel from peer")
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
	default:
		d.Logger.Warnf("Unknown message type received")
	}
}
