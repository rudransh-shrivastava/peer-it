package daemon

import (
	"github.com/pion/webrtc/v3"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
)

func (d *Daemon) handleTrackerMsgs() {
	d.Logger.Info("Connected to tracker server")
	for {
		select {
		case <-d.Ctx.Done():
			d.Logger.Info("Stopping the tracker message listener")
			return
		case signalingMsg := <-d.TrackerSignalingCh:

			peerID := signalingMsg.Signaling.GetSourcePeerId()
			pc, exists := d.PeerConnections[peerID]

			if !exists {
				// If we don't have a connection yet and this is an offer, create one
				if offer := signalingMsg.Signaling.GetOffer(); offer != nil {
					config := webrtc.Configuration{
						ICEServers: []webrtc.ICEServer{
							{
								URLs: []string{
									"stun:stun.l.google.com:19302",
									"stun:stun1.l.google.com:19302",
									"stun:stun2.l.google.com:19302",
									"stun:stun3.l.google.com:19302",
									"stun:stun4.l.google.com:19302",
								},
							},
						},
					}

					err := d.handleWebRTCConnection(peerID, "", config, false)

					if err != nil {
						d.Logger.Warnf("Failed to create peer connection: %v", err)
						continue
					}
					pc = d.PeerConnections[peerID]
				} else {
					d.Logger.Warn("Received signaling message for unknown peer")
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
					d.Logger.Warnf("Failed to set remote description: %v", err)
					continue
				}

				answer, err := pc.CreateAnswer(nil)
				if err != nil {
					d.Logger.Warnf("Failed to create answer: %v", err)
					continue
				}

				err = pc.SetLocalDescription(answer)
				if err != nil {
					d.Logger.Warnf("Failed to set local description: %v", err)
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

				err = d.TrackerRouter.WriteMessage(answerMsg)
				if err != nil {
					d.Logger.Warnf("Failed to send answer: %v", err)
				}

			case signalingMsg.Signaling.GetAnswer() != nil:
				answer := signalingMsg.Signaling.GetAnswer()

				err := pc.SetRemoteDescription(webrtc.SessionDescription{
					Type: webrtc.SDPTypeAnswer,
					SDP:  answer.GetSdp(),
				})
				if err != nil {
					d.Logger.Warnf("Failed to set remote description: %v", err)
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
					d.Logger.Warnf("Failed to add ICE candidate: %v", err)
				}
			}

		case peerlistResponse := <-d.TrackerPeerListResponseCh:
			d.Logger.Debugf("Received peer list response from tracker: %+v", peerlistResponse.PeerListResponse)
			channel, exists := d.PendingPeerListRequests[peerlistResponse.PeerListResponse.GetFileHash()]
			if !exists {
				d.Logger.Warnf("No Requests for file hash: %s", peerlistResponse.PeerListResponse.GetFileHash())
			}

			responseMsg := &protocol.NetworkMessage_PeerListResponse{
				PeerListResponse: peerlistResponse.PeerListResponse,
			}

			channel <- responseMsg
			d.Logger.Info("Sent peer list response to CLI")
		case idMsg := <-d.TrackerIdMessageCh:
			d.ID = idMsg.Id.GetId()
			d.Logger.Infof("Got my ID from tracker server: %s", d.ID)
		}
	}
}
