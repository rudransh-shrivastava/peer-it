package node

import (
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
)

func (n *Node) handleTrackerMsgs() {
	n.logger.Info("Connected to tracker server")
	for {
		select {
		case <-n.ctx.Done():
			n.logger.Info("Stopping the tracker message listener")
			return
		case signalingMsg := <-n.trackerSignalingCh:

			peerID := signalingMsg.Signaling.GetSourcePeerId()
			pc, exists := n.peerConnections[peerID]

			if !exists {
				// If we don't have a connection yet and this is an offer, create one
				if offer := signalingMsg.Signaling.GetOffer(); offer != nil {
					config := DefaultSTUNConfig()

					err := n.handleWebRTCConnection(peerID, "", config, false)

					if err != nil {
						n.logger.Warnf("Failed to create peer connection: %v", err)
						continue
					}
					pc = n.peerConnections[peerID]
				} else {
					n.logger.Warn("Received signaling message for unknown peer")
					continue
				}
			}

			switch {
			case signalingMsg.Signaling.GetOffer() != nil:
				offer := signalingMsg.Signaling.GetOffer()

				err := pc.SetRemoteDescription(webrtc.SessionDescription{
					Type: webrtc.SDPTypeOffer,
					SDP:  offer.GetSdp(),
				})
				if err != nil {
					n.logger.Warnf("Failed to set remote description: %v", err)
					continue
				}

				answer, err := pc.CreateAnswer(nil)
				if err != nil {
					n.logger.Warnf("Failed to create answer: %v", err)
					continue
				}

				err = pc.SetLocalDescription(answer)
				if err != nil {
					n.logger.Warnf("Failed to set local description: %v", err)
					continue
				}

				answerMsg := &protocol.NetworkMessage{
					MessageType: &protocol.NetworkMessage_Signaling{
						Signaling: &protocol.SignalingMessage{
							TargetPeerId: peerID,
							Message: &protocol.SignalingMessage_Answer{
								Answer: &protocol.Answer{
									Sdp: answer.SDP,
								},
							},
						},
					},
				}

				err = n.trackerRouter.WriteMessage(answerMsg)
				if err != nil {
					n.logger.Warnf("Failed to send answer: %v", err)
				}

			case signalingMsg.Signaling.GetAnswer() != nil:
				answer := signalingMsg.Signaling.GetAnswer()

				err := pc.SetRemoteDescription(webrtc.SessionDescription{
					Type: webrtc.SDPTypeAnswer,
					SDP:  answer.GetSdp(),
				})
				if err != nil {
					n.logger.Warnf("Failed to set remote description: %v", err)
				}

			case signalingMsg.Signaling.GetIceCandidate() != nil:
				ice := signalingMsg.Signaling.GetIceCandidate()

				var sdpmlineIndex = uint16(ice.GetSdpMlineIndex())
				err := pc.AddICECandidate(webrtc.ICECandidateInit{
					Candidate:     ice.GetCandidate(),
					SDPMid:        &ice.SdpMid,
					SDPMLineIndex: &sdpmlineIndex,
				})
				if err != nil {
					n.logger.Warnf("Failed to add ICE candidate: %v", err)
				}
			}

		case peerlistResponse := <-n.trackerPeerListResponseCh:
			n.logger.Debugf("Received peer list response from tracker: %+v", peerlistResponse.PeerListResponse)
			channel, exists := n.pendingPeerListRequests[peerlistResponse.PeerListResponse.GetFileHash()]
			if !exists {
				n.logger.Warnf("No Requests for file hash: %s", peerlistResponse.PeerListResponse.GetFileHash())
			}

			responseMsg := &protocol.NetworkMessage_PeerListResponse{
				PeerListResponse: peerlistResponse.PeerListResponse,
			}

			channel <- responseMsg
			n.logger.Info("Sent peer list response to CLI")
		case idMsg := <-n.trackerIDMessageCh:
			n.id = idMsg.Id.GetId()
			n.logger.Infof("Got my ID from tracker server: %s", n.id)
		}
	}
}

func (n *Node) sendHeartBeatsToTracker() {
	ticker := time.NewTicker(heartbeatInterval * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-n.ctx.Done():
			n.logger.Info("Stopping the heart")
			return
		case <-ticker.C:
			err := n.trackerRouter.WriteMessage(BuildHeartbeatMessage(time.Now().Unix()))
			if err != nil {
				n.logger.Warnf("Error sending message to Tracker: %v", err)
			}
		}
	}
}
