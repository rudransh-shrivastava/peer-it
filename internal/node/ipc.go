package node

import (
	"context"
	"net"
	"os"

	"github.com/rudransh-shrivastava/peer-it/internal/shared/protocol"
	"github.com/rudransh-shrivastava/peer-it/internal/shared/prouter"
	"google.golang.org/protobuf/proto"
)

func (n *Node) startIPCServer() {
	socketURL := BuildSocketPath(n.ipcSocketIndex)
	_ = os.Remove(socketURL)

	l, err := net.Listen("unix", socketURL)
	if err != nil {
		n.logger.Fatalf("Failed to start IPC server: %v", err)
		return
	}
	n.logger.Info("IPC Server started successfully")

	for {
		cliConn, err := l.Accept()
		n.logger.Info("Accepted a new socket connection")
		if err != nil {
			continue
		}

		cliRouter := prouter.NewMessageRouter(cliConn)
		cliRouter.AddRoute(n.cliSignalRegisterCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_SignalRegister)
			return ok
		})
		cliRouter.AddRoute(n.cliSignalDownloadCh, func(msg proto.Message) bool {
			_, ok := msg.(*protocol.NetworkMessage).MessageType.(*protocol.NetworkMessage_SignalDownload)
			return ok
		})

		channels := Channels{
			LogCh:     make(chan *protocol.NetworkMessage_Log, 100),
			GoodbyeCh: make(chan *protocol.NetworkMessage_Goodbye, 100),
		}
		n.channels[cliConn.RemoteAddr().String()] = channels
		n.cliRouter = cliRouter

		go cliRouter.Start()
		go n.handleCLIMsgs(cliRouter)
	}
}

func (n *Node) handleCLIMsgs(cliRouter *prouter.MessageRouter) {
	ctx := context.Background()
	cliAddr := cliRouter.Conn.RemoteAddr().String()

	for {
		select {
		case <-n.ctx.Done():
			n.logger.Info("Stopping the CLI message listener")
			return
		case downloadSignal := <-n.cliSignalDownloadCh:
			n.handleDownloadSignal(ctx, cliAddr, downloadSignal)
		case registerSignal := <-n.cliSignalRegisterCh:
			n.handleRegisterSignal(ctx, cliAddr, registerSignal)
		}
	}
}
