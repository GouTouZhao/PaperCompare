package cmd

import (
	"fmt"

	"github.com/GouTouZhao/PaperCompare/pkg/common/config"
	"github.com/GouTouZhao/PaperCompare/tools/errs"
	"github.com/spf13/cobra"
)

type RootCmd struct {
	Command     cobra.Command
	processName string
	port        int
	index       int
	configPath  string
}

func (r *RootCmd) ConfigPath() string {
	return r.configPath
}

func (r *RootCmd) Index() int {
	return r.index
}

func (r *RootCmd) Port() int {
	return r.port
}

type CmdOpts struct {
	configMap map[string]any
}

func WithConfigMap(configMap map[string]any) func(*CmdOpts) {
	return func(opts *CmdOpts) {
		opts.configMap = configMap
	}
}

func NewRootCmd(processName string, opts ...func(*CmdOpts)) *RootCmd {
	rootCmd := &RootCmd{
		processName: processName,
	}
	cmd := cobra.Command{
		Use:           "Start RPC application",
		Long:          fmt.Sprintf(`Start %s`, processName),
		SilenceUsage:  true,
		SilenceErrors: false,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return rootCmd.persistentPreRun(cmd, opts...)
		},
	}

	cmd.Flags().StringP(config.FlagConf, "c", "", "path of config directory")
	cmd.Flags().IntP(config.FlagTransferIndex, "i", 0, "process startup sequence number")

	rootCmd.Command = cmd
	return rootCmd
}

func (r *RootCmd) persistentPreRun(cmd *cobra.Command, opts ...func(*CmdOpts)) error {
	cmdOpts := r.applyOptions(opts...)
	return r.initializeConfiguration(cmd, cmdOpts)
}

func (r *RootCmd) initializeConfiguration(cmd *cobra.Command, opts *CmdOpts) error {
	configDirectory, _, err := r.getFlag(cmd)
	if err != nil {
		return err
	}
	for configFileName, configStruct := range opts.configMap {
		if err := config.Load(configDirectory, configFileName, configStruct); err != nil {
			return err
		}
	}
	return nil
}

func (r *RootCmd) applyOptions(opts ...func(*CmdOpts)) *CmdOpts {
	cmdOpts := &CmdOpts{}
	for _, opt := range opts {
		opt(cmdOpts)
	}
	return cmdOpts
}

func (r *RootCmd) getFlag(cmd *cobra.Command) (string, int, error) {
	configDirectory, err := cmd.Flags().GetString(config.FlagConf)
	if err != nil {
		return "", 0, errs.Wrap(err)
	}
	r.configPath = configDirectory
	index, err := cmd.Flags().GetInt(config.FlagTransferIndex)
	if err != nil {
		return "", 0, errs.Wrap(err)
	}
	r.index = index
	return configDirectory, index, nil
}

func (r *RootCmd) Execute() error {
	return r.Command.Execute()
}

