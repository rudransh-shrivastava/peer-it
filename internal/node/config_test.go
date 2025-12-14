package node

import (
	"testing"

	"github.com/pion/webrtc/v3"
)

func TestDefaultSTUNConfig(t *testing.T) {
	config := DefaultSTUNConfig()

	if len(config.ICEServers) != 1 {
		t.Errorf("expected 1 ICE server group, got %d", len(config.ICEServers))
	}

	if len(config.ICEServers[0].URLs) != 5 {
		t.Errorf("expected 5 STUN URLs, got %d", len(config.ICEServers[0].URLs))
	}

	if config.ICETransportPolicy != webrtc.ICETransportPolicyAll {
		t.Errorf("expected ICETransportPolicyAll")
	}
}

func TestDefaultDataChannelConfig(t *testing.T) {
	config := DefaultDataChannelConfig()

	if config.Ordered == nil || !*config.Ordered {
		t.Error("expected Ordered to be true")
	}

	if config.MaxRetransmits != nil {
		t.Error("expected MaxRetransmits to be nil (unlimited)")
	}

	if config.Protocol == nil || *config.Protocol != "file-transfer" {
		t.Error("expected Protocol to be 'file-transfer'")
	}
}
