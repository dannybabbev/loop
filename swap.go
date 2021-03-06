package loop

import (
	"context"
	"time"

	"github.com/lightninglabs/loop/lndclient"
	"github.com/lightninglabs/loop/loopdb"
	"github.com/lightninglabs/loop/swap"
	"github.com/lightningnetwork/lnd/lntypes"
)

type swapKit struct {
	htlc *swap.Htlc
	hash lntypes.Hash

	height int32

	log *swap.PrefixLog

	lastUpdateTime time.Time
	cost           loopdb.SwapCost
	state          loopdb.SwapState
	executeConfig
	swapConfig

	contract *loopdb.SwapContract
	swapType swap.Type
}

func newSwapKit(hash lntypes.Hash, swapType swap.Type, cfg *swapConfig,
	contract *loopdb.SwapContract, outputType swap.HtlcOutputType) (
	*swapKit, error) {

	// Compose expected on-chain swap script
	htlc, err := swap.NewHtlc(
		contract.CltvExpiry, contract.SenderKey,
		contract.ReceiverKey, hash, outputType,
		cfg.lnd.ChainParams,
	)
	if err != nil {
		return nil, err
	}

	log := &swap.PrefixLog{
		Hash:   hash,
		Logger: log,
	}

	// Log htlc address for debugging.
	log.Infof("Htlc address: %v", htlc.Address)

	return &swapKit{
		swapConfig: *cfg,
		hash:       hash,
		log:        log,
		htlc:       htlc,
		state:      loopdb.StateInitiated,
		contract:   contract,
		swapType:   swapType,
	}, nil
}

// sendUpdate reports an update to the swap state.
func (s *swapKit) sendUpdate(ctx context.Context) error {
	info := &SwapInfo{
		SwapContract: *s.contract,
		SwapHash:     s.hash,
		SwapType:     s.swapType,
		LastUpdate:   s.lastUpdateTime,
		SwapStateData: loopdb.SwapStateData{
			State: s.state,
			Cost:  s.cost,
		},
		HtlcAddress: s.htlc.Address,
	}

	s.log.Infof("state %v", info.State)

	select {
	case s.statusChan <- *info:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

type genericSwap interface {
	execute(mainCtx context.Context, cfg *executeConfig,
		height int32) error
}

type swapConfig struct {
	lnd    *lndclient.LndServices
	store  loopdb.SwapStore
	server swapServerClient
}
