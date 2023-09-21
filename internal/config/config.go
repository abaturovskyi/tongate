package config

import (
	"time"

	"github.com/caarlos0/env/v9"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

var Config config

type config struct {
	LiteServerConfigURL  string         `env:"LITESERVER_CFG_URL,required"`
	DefaultWalletVersion wallet.Version `env:"DEFAULT_WALLET_VERSION,required"`
	Seed                 string         `env:"SEED,required"`
	DatabaseURI          string         `env:"DB_URI,required"`
	APIPort              string         `env:"API_PORT,required"`
	AdminToken           string         `env:"ADMIN_TOKEN,required"`
	BlockTTL             time.Duration  `env:"BLOCK_TTL,default=10s"`
	LogOutputPath        string         `env:"LOG_OUTPUT_PATH,required"`
}

func init() {
	Config = config{}
	if err := env.Parse(&Config); err != nil {
		panic(err)
	}
}
