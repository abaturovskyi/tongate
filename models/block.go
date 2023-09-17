package models

import (
	"time"

	"github.com/abaturovskyi/tongate/config"
	"github.com/xssnick/tonutils-go/ton"
)

const Workchain = 0

type BlockHeader struct {
	*ton.BlockIDExt
	GenUtime uint32
	StartLt  uint64
	EndLt    uint64
	Parent   *ton.BlockIDExt
}

func (b *BlockHeader) IsExpired() bool {
	return time.Since(time.Unix(int64(b.GenUtime), 0)) > config.Config.BlockTTL
}
