package trading

import (
	"github.com/tonatos/instrumenta/backend/internal/domain/bonds"
	"github.com/tonatos/instrumenta/backend/internal/domain/trading"
)

// UniverseAugmenter keeps held bonds in the market universe when screener filters drop them.
type UniverseAugmenter interface {
	AugmentUniverseForBrokerSnapshot(universe []bonds.BondRecord, snapshot trading.BrokerSnapshot, keyRate, taxRate float64) []bonds.BondRecord
	AugmentUniverseForISINs(universe []bonds.BondRecord, isins []string, keyRate, taxRate float64) []bonds.BondRecord
}

func augmentUniverseForSnapshot(
	augmenter UniverseAugmenter,
	universe []bonds.BondRecord,
	snapshot trading.BrokerSnapshot,
	keyRate, taxRate float64,
) []bonds.BondRecord {
	if augmenter == nil || len(snapshot.BondPositions) == 0 {
		return universe
	}
	return augmenter.AugmentUniverseForBrokerSnapshot(universe, snapshot, keyRate, taxRate)
}
