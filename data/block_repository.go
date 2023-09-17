package data

import (
	"context"
	"database/sql"
	"errors"

	"github.com/abaturovskyi/tongate/models"
	"github.com/uptrace/bun"
)

type BlockRepository interface {
	GetLastBlock(ctx context.Context) (*models.BlockHeader, error)
}

type blockRepository struct {
	*useTx
}

func NewBlockRepository(client *bun.DB) BlockRepository {
	return &blockRepository{&useTx{client}}
}

func (c *blockRepository) GetLastBlock(ctx context.Context) (*models.BlockHeader, error) {
	var block = models.BlockHeader{}
	err := c.client.QueryRowContext(ctx, `
		SELECT
		    seqno,
		    shard,
		    root_hash,
		    file_hash,
				gen_utime
		FROM payments.block_header
		ORDER BY seqno DESC
		LIMIT 1
	`).Scan(
		&block,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	block.Workchain = models.Workchain
	return &block, nil
}
