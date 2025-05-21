package state

import (
	"fmt"

	abcitypes "github.com/cometbft/cometbft/abci/types"

	"github.com/cometbft/cometbft/state/indexer"
	"github.com/cometbft/cometbft/state/txindex"
	"github.com/cometbft/cometbft/types"
)

func IndexBlockAndTxsByNumber(blockStore BlockStore, stateStore Store, blockIndexer indexer.BlockIndexer, txIndexer txindex.TxIndexer, height int64) error {
	block := blockStore.LoadBlock(height)
	if block == nil {
		return fmt.Errorf("not able to load block at height %d from the blockstore", height)
	}

	resp, err := stateStore.LoadFinalizeBlockResponse(height)
	if err != nil {
		return fmt.Errorf("not able to load ABCI Response at height %d from the statestore", height)
	}

	e := types.EventDataNewBlockEvents{
		Height: height,
		Events: resp.Events,
	}

	numTxs := len(resp.TxResults)

	var batch *txindex.Batch
	if numTxs > 0 {
		batch = txindex.NewBatch(int64(numTxs))

		for idx, txResult := range resp.TxResults {
			tr := abcitypes.TxResult{
				Height: height,
				Index:  uint32(idx),
				Tx:     block.Txs[idx],
				Result: *txResult,
			}

			if err = batch.Add(&tr); err != nil {
				return fmt.Errorf("adding tx to batch: %w", err)
			}
		}

		if err := txIndexer.AddBatch(batch); err != nil {
			return fmt.Errorf("tx event re-index at height %d failed: %w", height, err)
		}
	}

	if err := blockIndexer.Index(e); err != nil {
		return fmt.Errorf("block event re-index at height %d failed: %w", height, err)
	}
	return nil
}
