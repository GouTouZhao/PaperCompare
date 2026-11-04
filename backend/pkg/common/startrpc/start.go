package startrpc

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/GouTouZhao/PaperCompare/pkg/common/config"
	discovery_pkg "github.com/GouTouZhao/PaperCompare/pkg/common/discovery"
	"github.com/GouTouZhao/PaperCompare/tools/discovery"
	"github.com/GouTouZhao/PaperCompare/tools/errs"

	"google.golang.org/grpc"
)

func Start[T any](
	ctx context.Context,
	disco *config.Discovery,
	listenIP string,
	registerIP string,
	rpcPorts int,
	index int,
	rpcRegisterName string,
	configt T,
	watchServiceName []string,
	rpcFn func(ctx context.Context, configt T, client discovery.SvcDiscoveryRegistry, server grpc.ServiceRegistrar) error,
) error {
	client, err := discovery_pkg.NewDiscoveryRegister(disco, watchServiceName)
	if err != nil {
		return errs.WrapMsg(err, "failed to create discovery register")
	}
	defer client.Close()

	ctx, cancel := context.WithCancelCause(ctx)

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
		select {
		case <-ctx.Done():
			return
		case val := <-sigs:
			cancel(fmt.Errorf("signal %s", val.String()))
		}
	}()

	var (
		rpcServer       *grpc.Server
		rpcGracefulStop chan struct{}
	)

	onGrpcServiceRegister := func(desc *grpc.ServiceDesc, impl any) {
		if rpcServer != nil {
			rpcServer.RegisterService(desc, impl)
			return
		}

		rpcPort := rpcPorts
		rpcListenAddr := net.JoinHostPort(listenIP, strconv.Itoa(rpcPort))
		rpcListener, err := net.Listen("tcp", rpcListenAddr)
		if err != nil {
			cancel(fmt.Errorf("listen %s %s %w", rpcRegisterName, rpcListenAddr, err))
			return
		}

		rpcServer = grpc.NewServer(
			grpc.MaxRecvMsgSize(64*1024*1024),
			grpc.MaxSendMsgSize(64*1024*1024),
		)
		rpcServer.RegisterService(desc, impl)
		rpcPort = rpcListener.Addr().(*net.TCPAddr).Port
		rpcGracefulStop = make(chan struct{})

		go func() {
			<-ctx.Done()
			rpcServer.GracefulStop()
			close(rpcGracefulStop)
		}()

		if err := client.Register(ctx, rpcRegisterName, registerIP, rpcPort); err != nil {
			cancel(fmt.Errorf("rpc register %s %w", rpcRegisterName, err))
			return
		}

		fmt.Printf("[RPC] %s listening on %s\n", rpcRegisterName, rpcListenAddr)
		go func() {
			err := rpcServer.Serve(rpcListener)
			if err != nil {
				err = fmt.Errorf("serve end: %w", err)
			}
			cancel(fmt.Errorf("rpc %s %w", rpcRegisterName, err))
		}()
	}

	err = rpcFn(ctx, configt, client, &grpcServiceRegistrar{onRegisterService: onGrpcServiceRegister})
	if err != nil {
		return errs.WrapMsg(err, "rpcFn() wrong")
	}

	<-ctx.Done()
	if rpcGracefulStop != nil {
		timeout := time.NewTimer(time.Second * 3)
		defer timeout.Stop()
		select {
		case <-timeout.C:
			fmt.Println("rpc graceful stop timeout")
		case <-rpcGracefulStop:
			fmt.Println("rpc graceful stop done")
		}
	}
	return context.Cause(ctx)
}

type grpcServiceRegistrar struct {
	onRegisterService func(desc *grpc.ServiceDesc, impl any)
}

func (x *grpcServiceRegistrar) RegisterService(desc *grpc.ServiceDesc, impl any) {
	x.onRegisterService(desc, impl)
}

