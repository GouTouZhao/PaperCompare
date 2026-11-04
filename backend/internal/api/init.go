package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/GouTouZhao/PaperCompare/pkg/common/config"
	"github.com/GouTouZhao/PaperCompare/tools/discovery"
	"github.com/GouTouZhao/PaperCompare/tools/errs"
	"google.golang.org/grpc"
)

type Config struct {
	config.ALLConfig
	ConfigPath config.Path
	Index      config.Index
}

func Start(ctx context.Context, configStart *Config, client discovery.SvcDiscoveryRegistry, server grpc.ServiceRegistrar) error {
	if envNode := os.Getenv("NODE_NAME"); envNode != "" {
		configStart.API.Api.NodeName = envNode
	}
	if envURL := os.Getenv("NODE_PUBLIC_URL"); envURL != "" {
		configStart.API.Api.PublicURL = envURL
	}
	apiPort := 10002
	if len(configStart.API.Api.Ports) > 0 && configStart.API.Api.Ports[0] > 0 {
		apiPort = configStart.API.Api.Ports[0]
	}
	if apiPort == 0 {
		return errs.ErrInternalServer.WrapMsg("api service's port is nil")
	}

	router, err := newGinRouter(ctx, client, configStart)
	if err != nil {
		return err
	}

	apiCtx, apiCancel := context.WithCancelCause(context.Background())
	done := make(chan struct{})

	go func() {
		httpServer := &http.Server{
			Handler: router,
			Addr:    net.JoinHostPort(configStart.API.Api.ListenIP, strconv.Itoa(apiPort)),
		}
		go func() {
			defer close(done)
			select {
			case <-ctx.Done():
				apiCancel(fmt.Errorf("recv ctx %w", context.Cause(ctx)))
			case <-apiCtx.Done():
			}
			fmt.Println("api server is shutting down")
			_ = httpServer.Shutdown(context.Background())
		}()
		fmt.Printf("[API] PaperCompare Gateway listening on %s\n", httpServer.Addr)
		err := httpServer.ListenAndServe()
		if err != nil {
			err = errors.New("api done")
		}
		apiCancel(err)
	}()

	<-apiCtx.Done()
	exitCause := context.Cause(apiCtx)
	timer := time.NewTimer(time.Second * 5)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-done:
	}
	return exitCause
}
