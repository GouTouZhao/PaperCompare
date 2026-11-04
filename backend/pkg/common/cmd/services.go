package cmd

import (
	"context"

	"github.com/GouTouZhao/PaperCompare/internal/api"
	"github.com/GouTouZhao/PaperCompare/internal/rpc/compare"
	"github.com/GouTouZhao/PaperCompare/internal/rpc/paper"
	"github.com/GouTouZhao/PaperCompare/internal/rpc/user"
	"github.com/GouTouZhao/PaperCompare/pkg/common/config"
	"github.com/GouTouZhao/PaperCompare/pkg/common/startrpc"
	"github.com/GouTouZhao/PaperCompare/version"
	"github.com/spf13/cobra"
)

// ===== API Gateway Cmd =====

type ApiCmd struct {
	*RootCmd
	ctx       context.Context
	configMap map[string]any
	apiConfig *api.Config
}

func NewApiCmd() *ApiCmd {
	var apiConfig api.Config
	ret := &ApiCmd{apiConfig: &apiConfig}
	ret.configMap = map[string]any{
		config.DiscoveryConfigFilename: &apiConfig.Discovery,
		config.APIConfigFilename:       &apiConfig.API,
		config.StorageConfigFilename:   &apiConfig.Storage,
	}
	ret.RootCmd = NewRootCmd("paper-api", WithConfigMap(ret.configMap))
	ret.ctx = context.WithValue(context.Background(), "version", version.Version)
	ret.Command.RunE = func(cmd *cobra.Command, args []string) error {
		apiConfig.ConfigPath = config.Path(ret.configPath)
		return startrpc.Start(
			ret.ctx,
			&ret.apiConfig.Discovery,
			ret.apiConfig.API.Api.ListenIP,
			"",
			ret.apiConfig.API.Api.Ports[0],
			0,
			ret.apiConfig.Discovery.RpcService.API,
			ret.apiConfig,
			[]string{},
			api.Start,
		)
	}
	return ret
}

func (a *ApiCmd) Exec() error { return a.Execute() }

// ===== User RPC Cmd =====

type UserRpcCmd struct {
	*RootCmd
	ctx        context.Context
	configMap  map[string]any
	userConfig *user.Config
}

func NewRpcUserCmd() *UserRpcCmd {
	var cfg user.Config
	ret := &UserRpcCmd{userConfig: &cfg}
	ret.configMap = map[string]any{
		config.DiscoveryConfigFilename: &cfg.Discovery,
		config.RPCUserConfigFilename:   &cfg.User,
		config.PostgresConfigFilename:  &cfg.Postgres,
		config.RedisConfigFilename:     &cfg.Redis,
	}
	ret.RootCmd = NewRootCmd("paper-rpc-user", WithConfigMap(ret.configMap))
	ret.ctx = context.WithValue(context.Background(), "version", version.Version)
	ret.Command.RunE = func(cmd *cobra.Command, args []string) error {
		cfg.ConfigPath = config.Path(ret.configPath)
		return startrpc.Start(
			ret.ctx,
			&cfg.Discovery,
			cfg.User.RPC.ListenIP,
			"",
			cfg.User.RPC.Port,
			0,
			cfg.Discovery.RpcService.User,
			&cfg,
			[]string{},
			user.Start,
		)
	}
	return ret
}

func (a *UserRpcCmd) Exec() error { return a.Execute() }

// ===== Paper RPC Cmd =====

type PaperRpcCmd struct {
	*RootCmd
	ctx         context.Context
	configMap   map[string]any
	paperConfig *paper.Config
}

func NewRpcPaperCmd() *PaperRpcCmd {
	var cfg paper.Config
	ret := &PaperRpcCmd{paperConfig: &cfg}
	ret.configMap = map[string]any{
		config.DiscoveryConfigFilename: &cfg.Discovery,
		config.RPCPaperConfigFilename:  &cfg.Paper,
		config.PostgresConfigFilename:  &cfg.Postgres,
		config.LLMConfigFilename:       &cfg.LLM,
		config.StorageConfigFilename:   &cfg.Storage,
	}
	ret.RootCmd = NewRootCmd("paper-rpc-paper", WithConfigMap(ret.configMap))
	ret.ctx = context.WithValue(context.Background(), "version", version.Version)
	ret.Command.RunE = func(cmd *cobra.Command, args []string) error {
		cfg.ConfigPath = config.Path(ret.configPath)
		return startrpc.Start(
			ret.ctx,
			&cfg.Discovery,
			cfg.Paper.RPC.ListenIP,
			"",
			cfg.Paper.RPC.Port,
			0,
			cfg.Discovery.RpcService.Paper,
			&cfg,
			[]string{},
			paper.Start,
		)
	}
	return ret
}

func (a *PaperRpcCmd) Exec() error { return a.Execute() }

// ===== Compare RPC Cmd =====

type CompareRpcCmd struct {
	*RootCmd
	ctx           context.Context
	configMap     map[string]any
	compareConfig *compare.Config
}

func NewRpcCompareCmd() *CompareRpcCmd {
	var cfg compare.Config
	ret := &CompareRpcCmd{compareConfig: &cfg}
	ret.configMap = map[string]any{
		config.DiscoveryConfigFilename:  &cfg.Discovery,
		config.RPCCompareConfigFilename: &cfg.Compare,
		config.PostgresConfigFilename:   &cfg.Postgres,
		config.LLMConfigFilename:        &cfg.LLM,
	}
	ret.RootCmd = NewRootCmd("paper-rpc-compare", WithConfigMap(ret.configMap))
	ret.ctx = context.WithValue(context.Background(), "version", version.Version)
	ret.Command.RunE = func(cmd *cobra.Command, args []string) error {
		cfg.ConfigPath = config.Path(ret.configPath)
		return startrpc.Start(
			ret.ctx,
			&cfg.Discovery,
			cfg.Compare.RPC.ListenIP,
			"",
			cfg.Compare.RPC.Port,
			0,
			cfg.Discovery.RpcService.Compare,
			&cfg,
			[]string{},
			compare.Start,
		)
	}
	return ret
}

func (a *CompareRpcCmd) Exec() error { return a.Execute() }
