package node

import "github.com/pion/webrtc/v3"

var defaultSTUNServers = []string{
	"stun:stun.l.google.com:19302",
	"stun:stun1.l.google.com:19302",
	"stun:stun2.l.google.com:19302",
	"stun:stun3.l.google.com:19302",
	"stun:stun4.l.google.com:19302",
}

func DefaultSTUNConfig() webrtc.Configuration {
	return webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: defaultSTUNServers},
		},
		ICETransportPolicy: webrtc.ICETransportPolicyAll,
	}
}

func DefaultDataChannelConfig() *webrtc.DataChannelInit {
	protocolName := "file-transfer"
	ordered := true
	return &webrtc.DataChannelInit{
		Ordered:        &ordered,
		MaxRetransmits: nil,
		Protocol:       &protocolName,
	}
}
