package direct

import (
	"context"
	"os"
	"strings"
	"sync"

	"github.com/GouTouZhao/PaperCompare/tools/errs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ConnManager struct {
	mu          sync.RWMutex
	endpoints   map[string]string
	conns       map[string]*grpc.ClientConn
	dialOptions []grpc.DialOption
	selfTarget  string
}

func NewConnManager(endpoints map[string]string) (*ConnManager, error) {
	ep := make(map[string]string, len(endpoints))
	for k, v := range endpoints {
		ep[strings.ToLower(k)] = v
		ep[k] = v
	}
	if commEndpoint := os.Getenv("COMMUNITY_RPC_ENDPOINT"); commEndpoint != "" {
		ep["paper-community-service"] = commEndpoint
	}
	return &ConnManager{
		endpoints: ep,
		conns:     make(map[string]*grpc.ClientConn),
	}, nil
}

func (m *ConnManager) resolveTarget(serviceName string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if addr, ok := m.endpoints[serviceName]; ok && addr != "" {
		return addr
	}
	if addr, ok := m.endpoints[strings.ToLower(serviceName)]; ok && addr != "" {
		return addr
	}
	return serviceName
}

func (m *ConnManager) GetConn(ctx context.Context, serviceName string, opts ...grpc.DialOption) (grpc.ClientConnInterface, error) {
	target := m.resolveTarget(serviceName)

	m.mu.Lock()
	defer m.mu.Unlock()

	if c, ok := m.conns[target]; ok {
		return c, nil
	}

	dialOpts := append([]grpc.DialOption{}, m.dialOptions...)
	dialOpts = append(dialOpts, opts...)
	dialOpts = append(dialOpts,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(64*1024*1024),
			grpc.MaxCallSendMsgSize(64*1024*1024),
		),
	)

	conn, err := grpc.NewClient(target, dialOpts...)
	if err != nil {
		return nil, errs.WrapMsg(err, "failed to dial grpc target", "target", target)
	}
	m.conns[target] = conn
	return conn, nil
}

func (m *ConnManager) GetConns(ctx context.Context, serviceName string, opts ...grpc.DialOption) ([]grpc.ClientConnInterface, error) {
	conn, err := m.GetConn(ctx, serviceName, opts...)
	if err != nil {
		return nil, err
	}
	return []grpc.ClientConnInterface{conn}, nil
}

func (m *ConnManager) IsSelfNode(cc grpc.ClientConnInterface) bool {
	cli, ok := cc.(*grpc.ClientConn)
	if !ok {
		return false
	}
	return m.selfTarget != "" && m.selfTarget == cli.Target()
}

func (m *ConnManager) AddOption(opts ...grpc.DialOption) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dialOptions = append(m.dialOptions, opts...)
}

func (m *ConnManager) Register(ctx context.Context, serviceName, host string, port int, opts ...grpc.DialOption) error {
	return nil
}

func (m *ConnManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.conns {
		_ = c.Close()
	}
	m.conns = make(map[string]*grpc.ClientConn)
}
