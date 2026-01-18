package tracker

import (
	"testing"
)

func TestParseAddrToPeerInfo(t *testing.T) {
	tests := []struct {
		addr     string
		wantOk   bool
		wantIP   [16]byte
		wantPort uint16
	}{
		{
			addr:     "192.168.1.100:5000",
			wantOk:   true,
			wantIP:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff, 192, 168, 1, 100},
			wantPort: 5000,
		},
		{
			addr:     "[::1]:8080",
			wantOk:   true,
			wantIP:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			wantPort: 8080,
		},
		{
			addr:   "invalid",
			wantOk: false,
		},
		{
			addr:   "192.168.1.1:notaport",
			wantOk: false,
		},
		{
			addr:   "notanip:5000",
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			info, ok := parseAddrToPeerInfo(tt.addr)
			if ok != tt.wantOk {
				t.Errorf("parseAddrToPeerInfo(%q) ok = %v, want %v", tt.addr, ok, tt.wantOk)
				return
			}
			if !ok {
				return
			}
			if info.IP != tt.wantIP {
				t.Errorf("IP = %v, want %v", info.IP, tt.wantIP)
			}
			if info.Port != tt.wantPort {
				t.Errorf("Port = %d, want %d", info.Port, tt.wantPort)
			}
		})
	}
}
