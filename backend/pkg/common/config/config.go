package config

type Path string
type Index int

const (
	FlagConf          = "config_path"
	FlagTransferIndex = "index"

	MountConfigFilePath = "CONFIG_PATH"

	DiscoveryConfigFilename = "discovery.yaml"
	PostgresConfigFilename  = "postgres.yaml"
	RedisConfigFilename     = "redis.yaml"
	LLMConfigFilename       = "llm.yaml"
	StorageConfigFilename   = "storage.yaml"

	APIConfigFilename        = "paper-api.yaml"
	RPCUserConfigFilename    = "paper-rpc-user.yaml"
	RPCPaperConfigFilename   = "paper-rpc-paper.yaml"
	RPCCompareConfigFilename = "paper-rpc-compare.yaml"
)

type ALLConfig struct {
	Discovery Discovery
	Postgres  Postgres
	Redis     Redis
	LLM       LLM
	Storage   Storage
	API       API
	User      RPCServiceConfig
	Paper     RPCServiceConfig
	Compare   RPCServiceConfig
}

type Discovery struct {
	Enable     string            `yaml:"enable"`
	Endpoints  map[string]string `yaml:"endpoints"`
	RpcService RpcService        `yaml:"rpcService"`
}

type RpcService struct {
	API     string `yaml:"api"`
	User    string `yaml:"user"`
	Paper   string `yaml:"paper"`
	Compare string `yaml:"compare"`
}

type Postgres struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type Redis struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DBName   int    `yaml:"dbname"`
}

type LLM struct {
	Provider string `yaml:"provider"`
	BaseURL  string `yaml:"baseURL"`
	Model    string `yaml:"model"`
	APIKey   string `yaml:"apiKey"`
}

type Storage struct {
	DataDir string `yaml:"dataDir"`
	MaxSize int64  `yaml:"maxSize"`
}

type API struct {
	Api struct {
		ListenIP         string `yaml:"listenIP"`
		Ports            []int  `yaml:"ports"`
		CompressionLevel int    `yaml:"compressionLevel"`
		NodeName         string `yaml:"nodeName"`
		PublicURL        string `yaml:"publicURL"`
	} `yaml:"api"`
}

type RPC struct {
	RegisterIP string `yaml:"registerIP"`
	ListenIP   string `yaml:"listenIP"`
	Port       int    `yaml:"port"`
}

type RPCServiceConfig struct {
	RPC RPC `yaml:"rpc"`
}
