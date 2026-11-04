package discovery

import (
	"context"

	"google.golang.org/grpc"
)

type Conn interface {
	GetConn(ctx context.Context, serviceName string, opts ...grpc.DialOption) (grpc.ClientConnInterface, error)
	GetConns(ctx context.Context, serviceName string, opts ...grpc.DialOption) ([]grpc.ClientConnInterface, error)
	IsSelfNode(cc grpc.ClientConnInterface) bool
}

type SvcDiscoveryRegistry interface {
	Conn
	AddOption(opts ...grpc.DialOption)
	Register(ctx context.Context, serviceName, host string, port int, opts ...grpc.DialOption) error
	Close()
}

