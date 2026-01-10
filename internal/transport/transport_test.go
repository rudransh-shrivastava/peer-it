package transport

import (
	"context"
	"testing"
	"time"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
)

func TestTransportCreateAndClose(t *testing.T) {
	tr, err := NewTransport(":0")
	if err != nil {
		t.Fatalf("NewTransport failed: %v", err)
	}
	defer func() { _ = tr.Close() }()

	addr := tr.LocalAddr()
	if addr == nil {
		t.Error("Expected non-nil local address")
	}
}

func TestTransportDialAccept(t *testing.T) {
	server, err := NewTransport(":0")
	if err != nil {
		t.Fatalf("NewTransport server failed: %v", err)
	}
	defer func() { _ = server.Close() }()

	client, err := NewTransport(":0")
	if err != nil {
		t.Fatalf("NewTransport client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverAddr := server.LocalAddr().String()

	accepted := make(chan *Peer, 1)
	errChan := make(chan error, 1)

	go func() {
		peer, err := server.Accept(ctx)
		if err != nil {
			errChan <- err
			return
		}
		accepted <- peer
	}()

	clientPeer, err := client.Dial(ctx, serverAddr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer func() { _ = clientPeer.Close() }()

	select {
	case serverPeer := <-accepted:
		defer func() { _ = serverPeer.Close() }()
		if serverPeer.RemoteAddr() == "" {
			t.Error("Expected non-empty remote address")
		}
	case err := <-errChan:
		t.Fatalf("Accept failed: %v", err)
	case <-ctx.Done():
		t.Fatal("Timeout waiting for connection")
	}
}

func TestPeerSendReceive(t *testing.T) {
	server, err := NewTransport(":0")
	if err != nil {
		t.Fatalf("NewTransport server failed: %v", err)
	}
	defer func() { _ = server.Close() }()

	client, err := NewTransport(":0")
	if err != nil {
		t.Fatalf("NewTransport client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverAddr := server.LocalAddr().String()

	received := make(chan protocol.Message, 1)
	errChan := make(chan error, 1)

	go func() {
		peer, err := server.Accept(ctx)
		if err != nil {
			errChan <- err
			return
		}
		defer func() { _ = peer.Close() }()

		msg, err := peer.Receive(ctx)
		if err != nil {
			errChan <- err
			return
		}
		received <- msg
	}()

	clientPeer, err := client.Dial(ctx, serverAddr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer func() { _ = clientPeer.Close() }()

	err = clientPeer.Send(ctx, &protocol.Ping{})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	select {
	case msg := <-received:
		if _, ok := msg.(*protocol.Ping); !ok {
			t.Errorf("Expected *Ping, got %T", msg)
		}
	case err := <-errChan:
		t.Fatalf("Receive failed: %v", err)
	case <-ctx.Done():
		t.Fatal("Timeout waiting for message")
	}
}

func TestPeerBidirectionalExchange(t *testing.T) {
	server, err := NewTransport(":0")
	if err != nil {
		t.Fatalf("NewTransport server failed: %v", err)
	}
	defer func() { _ = server.Close() }()

	client, err := NewTransport(":0")
	if err != nil {
		t.Fatalf("NewTransport client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverAddr := server.LocalAddr().String()

	errChan := make(chan error, 1)
	clientDone := make(chan struct{})

	go func() {
		peer, err := server.Accept(ctx)
		if err != nil {
			errChan <- err
			return
		}
		defer func() { _ = peer.Close() }()

		msg, err := peer.Receive(ctx)
		if err != nil {
			errChan <- err
			return
		}

		if _, ok := msg.(*protocol.FileListReq); !ok {
			errChan <- err
			return
		}

		err = peer.Send(ctx, &protocol.FileListRes{
			Files: []protocol.FileEntry{
				{Name: "test.txt", Size: 1024},
			},
		})
		if err != nil {
			errChan <- err
			return
		}

		<-clientDone
	}()

	clientPeer, err := client.Dial(ctx, serverAddr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer func() { _ = clientPeer.Close() }()

	err = clientPeer.Send(ctx, &protocol.FileListReq{})
	if err != nil {
		t.Fatalf("Send FileListReq failed: %v", err)
	}

	msg, err := clientPeer.Receive(ctx)
	if err != nil {
		t.Fatalf("Receive FileListRes failed: %v", err)
	}
	close(clientDone)

	res, ok := msg.(*protocol.FileListRes)
	if !ok {
		t.Fatalf("Expected *FileListRes, got %T", msg)
	}

	if len(res.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(res.Files))
	}

	if res.Files[0].Name != "test.txt" {
		t.Errorf("Expected 'test.txt', got '%s'", res.Files[0].Name)
	}

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("Server error: %v", err)
		}
	default:
	}
}

func TestGenerateSelfSignedCert(t *testing.T) {
	cert, err := GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert failed: %v", err)
	}

	if len(cert.Certificate) == 0 {
		t.Error("Expected non-empty certificate")
	}
}
