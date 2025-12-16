package node

import (
	"context"

	"github.com/sirupsen/logrus"
)

type Node struct {
	ctx context.Context
	NodeParams
}

type NodeParams struct {
	ID          string
	Logger      *logrus.Logger
	TrackerAddr string
}

func NewNode(ctx context.Context, nodeParams NodeParams) *Node {
	return &Node{
		ctx,
		nodeParams,
	}
}
