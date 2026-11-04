package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/GouTouZhao/PaperCompare/tools/errs"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

func Load(configDirectory string, configFilename string, cfg any) error {
	dir := configDirectory
	if dir == "" {
		dir = os.Getenv(MountConfigFilePath)
	}
	if dir == "" {
		dir = "./config"
	}

	fullPath := filepath.Join(dir, configFilename)
	return loadConfig(fullPath, cfg)
}

func loadConfig(path string, cfg any) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return errs.WrapMsg(err, "failed to read config file", "path", path)
	}

	if err := v.Unmarshal(cfg, func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "yaml"
	}); err != nil {
		return errs.WrapMsg(err, "failed to unmarshal config", "path", path)
	}

	return nil
}

