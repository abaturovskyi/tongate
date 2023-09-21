package service

import (
	"context"
	"math/bits"
	"strings"
	"time"

	"github.com/abaturovskyi/tongate/internal/bus"
	"github.com/abaturovskyi/tongate/models"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"go.uber.org/zap"
)

const ErrBlockNotApplied = "block is not applied"

type ShardScanner struct {
	logger              *zap.SugaredLogger
	tonApi              *ton.APIClient
	shard               byte
	lastKnownShardBlock *ton.BlockIDExt
	lastMasterBlock     *ton.BlockIDExt
}

// NewShardScanner creates new tracker to get blocks with specific shard attribute.
func NewShardScanner(
	logger *zap.SugaredLogger,
	shard byte,
	tonApi *ton.APIClient,
) *ShardScanner {
	t := &ShardScanner{
		logger: logger,
		tonApi: tonApi,
		shard:  shard,
	}
	return t
}

// Start scans for blocks.
func (s *ShardScanner) Start(ctx context.Context, startBlock *ton.BlockIDExt) {
	// The interval between blocks can be up to 40 seconds.
	ctx = s.tonApi.Client().StickyContext(ctx)
	s.lastKnownShardBlock = startBlock

	for {
		masterBlock, err := s.getCurrentMasterBlock(ctx)
		if err != nil {
			s.logger.Errorf("getNextMasterBlockID err - %v", err)

			continue
		}
		err = s.loadShardBlocksBatch(ctx, masterBlock)
		if err != nil {
			s.logger.Errorf("loadShardBlocksBatch err - %v", err)

			continue
		}
	}
}

func (s *ShardScanner) getCurrentMasterBlock(ctx context.Context) (*ton.BlockIDExt, error) {
	for {
		masterBlock, err := s.tonApi.GetMasterchainInfo(ctx)
		if err != nil {
			// exit by context timeout
			return nil, err
		}
		if s.lastMasterBlock == nil {
			s.lastMasterBlock = masterBlock
			return masterBlock, nil
		}
		if masterBlock.SeqNo == s.lastMasterBlock.SeqNo {
			time.Sleep(time.Second * 30)
			continue
		}
		s.lastMasterBlock = masterBlock
		return masterBlock, nil
	}
}

func (s *ShardScanner) loadShardBlocksBatch(ctx context.Context, masterBlock *ton.BlockIDExt) error {
	var (
		blocksShardsInfo []*ton.BlockIDExt
		err              error
	)
	for {
		blocksShardsInfo, err = s.tonApi.GetBlockShardsInfo(ctx, masterBlock)
		if err != nil && isNotReadyError(err) { // TODO: clarify error type
			time.Sleep(time.Second)
			continue
		} else if err != nil {
			return err
		}
		break
	}
	err = s.handleShardBlocks(ctx, filterByShard(blocksShardsInfo, s.shard))
	if err != nil {
		return err
	}

	return nil
}

func (s *ShardScanner) handleShardBlocks(ctx context.Context, i *ton.BlockIDExt) error {
	var currentBlock *ton.BlockIDExt = i
	start := time.Now()

	var diff = int(i.SeqNo - s.lastKnownShardBlock.SeqNo)

	s.logger.Infof("Shard tracker. Seqno diff: %v", diff)

	for {
		isKnown := (s.lastKnownShardBlock.Shard == currentBlock.Shard) && (s.lastKnownShardBlock.SeqNo == currentBlock.SeqNo)
		if isKnown {
			s.lastKnownShardBlock = i
			break
		}

		h, err := s.getBlockHeader(ctx, currentBlock, s.shard)

		if err != nil {
			return err
		}

		if err := bus.EmitEvent(bus.TopicBlockFound, &h); err != nil {
			return err
		}

		currentBlock = h.Parent
	}

	s.logger.Infof("Shard tracker. Blocks processed: %v Elapsed time: %v sec", diff, time.Since(start).Seconds())

	return nil
}

// get shard block header for specific shard attribute with one parent
func (s *ShardScanner) getBlockHeader(ctx context.Context, shardBlockInfo *ton.BlockIDExt, shard byte) (models.BlockHeader, error) {
	var (
		err   error
		block *tlb.Block
	)
	for {
		block, err = s.tonApi.GetBlockData(ctx, shardBlockInfo)
		if err != nil && isNotReadyError(err) {
			continue
		} else if err != nil {
			return models.BlockHeader{}, err
			// exit by context timeout
		}
		break
	}
	return convertBlockToHeader(block, shardBlockInfo, shard)
}

func convertBlockToHeader(block *tlb.Block, info *ton.BlockIDExt, shard byte) (models.BlockHeader, error) {
	parents, err := block.BlockInfo.GetParentBlocks()
	if err != nil {
		return models.BlockHeader{}, nil
	}
	parent := filterByShard(parents, shard)
	return models.BlockHeader{
		GenUtime:   block.BlockInfo.GenUtime,
		StartLt:    block.BlockInfo.StartLt,
		EndLt:      block.BlockInfo.EndLt,
		Parent:     parent,
		BlockIDExt: info,
	}, nil
}

func filterByShard(headers []*ton.BlockIDExt, shard byte) *ton.BlockIDExt {
	for _, h := range headers {
		if isInShard(uint64(h.Shard), shard) {
			return h
		}
	}
	return nil
}

func isInShard(blockShardPrefix uint64, shard byte) bool {
	if blockShardPrefix == 0 {
		return false
	}
	prefixLen := 64 - 1 - bits.TrailingZeros64(blockShardPrefix) // without one insignificant bit
	if prefixLen > 8 {
		return false
	}
	res := (uint64(shard) << (64 - 8)) ^ blockShardPrefix

	return bits.LeadingZeros64(res) >= prefixLen
}

func isNotReadyError(err error) bool {
	return strings.Contains(err.Error(), ErrBlockNotApplied)
}
