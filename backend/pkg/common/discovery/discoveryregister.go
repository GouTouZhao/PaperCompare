package discovery_pkg

import (
	"os"
	"strings"

	"github.com/GouTouZhao/PaperCompare/pkg/common/config"
	"github.com/GouTouZhao/PaperCompare/tools/discovery"
	"github.com/GouTouZhao/PaperCompare/tools/discovery/direct"
)

func NewDiscoveryRegister(discoveryConfig *config.Discovery, watchNames []string) (discovery.SvcDiscoveryRegistry, error) {
	endpoints := make(map[string]string)
	for k, v := range discoveryConfig.Endpoints {
		endpoints[k] = v
	}

	overrideEnv := map[string]string{
		discoveryConfig.RpcService.User:    os.Getenv("RPC_USER_ADDR"),
		discoveryConfig.RpcService.Paper:   os.Getenv("RPC_PAPER_ADDR"),
		discoveryConfig.RpcService.Compare: os.Getenv("RPC_COMPARE_ADDR"),
	}
	for svc, addr := range overrideEnv {
		if svc != "" && strings.TrimSpace(addr) != "" {
			endpoints[svc] = strings.TrimSpace(addr)
		}
	}

	return direct.NewConnManager(endpoints)
}
