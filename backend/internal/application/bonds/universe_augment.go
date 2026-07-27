package bonds

import (
	"github.com/tonatos/instrumenta/backend/internal/domain/bonds"
	"github.com/tonatos/instrumenta/backend/internal/domain/trading"
	infraBonds "github.com/tonatos/instrumenta/backend/internal/infrastructure/bonds"
)

// MergeBondUniverse returns base bonds plus extra entries not already present by ISIN.
func MergeBondUniverse(base, extra []bonds.BondRecord) []bonds.BondRecord {
	if len(extra) == 0 {
		return base
	}
	byISIN := make(map[string]bonds.BondRecord, len(base)+len(extra))
	order := make([]string, 0, len(base)+len(extra))
	for _, b := range base {
		if b.ISIN == "" {
			continue
		}
		if _, ok := byISIN[b.ISIN]; !ok {
			order = append(order, b.ISIN)
		}
		byISIN[b.ISIN] = b
	}
	for _, b := range extra {
		if b.ISIN == "" {
			continue
		}
		if _, ok := byISIN[b.ISIN]; !ok {
			order = append(order, b.ISIN)
		}
		byISIN[b.ISIN] = b
	}
	out := make([]bonds.BondRecord, 0, len(order))
	for _, isin := range order {
		out = append(out, infraBonds.CloneBondRecord(byISIN[isin]))
	}
	return out
}

// AugmentUniverseForBrokerSnapshot keeps held broker positions in the market universe
// even when MOEX screener filters would drop them (e.g. on/after maturity day).
func (s *Service) AugmentUniverseForBrokerSnapshot(
	universe []bonds.BondRecord,
	snapshot trading.BrokerSnapshot,
	keyRate, taxRate float64,
) []bonds.BondRecord {
	if len(snapshot.BondPositions) == 0 {
		return universe
	}
	knownFIGI := make(map[string]struct{}, len(universe))
	for _, b := range universe {
		if b.FIGI != "" {
			knownFIGI[b.FIGI] = struct{}{}
		}
	}
	var extra []bonds.BondRecord
	for figi, pos := range snapshot.BondPositions {
		if pos.Lots <= 0 {
			continue
		}
		if _, ok := knownFIGI[figi]; ok {
			continue
		}
		if pos.Ticker == "" {
			continue
		}
		bond, err := s.moex.FetchHeldBondBySecid(pos.Ticker)
		if err != nil || bond == nil {
			continue
		}
		if bond.FIGI == "" {
			bond.FIGI = figi
		}
		scored := s.enrichFetchedBonds([]bonds.BondRecord{*bond}, keyRate, taxRate)
		extra = append(extra, scored...)
	}
	return MergeBondUniverse(universe, extra)
}

// AugmentUniverseForISINs ensures portfolio position ISINs stay in the universe.
func (s *Service) AugmentUniverseForISINs(
	universe []bonds.BondRecord,
	isins []string,
	keyRate, taxRate float64,
) []bonds.BondRecord {
	if len(isins) == 0 {
		return universe
	}
	known := make(map[string]struct{}, len(universe))
	for _, b := range universe {
		if b.ISIN != "" {
			known[b.ISIN] = struct{}{}
		}
	}
	missing := make(map[string]struct{})
	for _, isin := range isins {
		if isin == "" {
			continue
		}
		if _, ok := known[isin]; !ok {
			missing[isin] = struct{}{}
		}
	}
	if len(missing) == 0 {
		return universe
	}
	fetched, _ := s.moex.FetchHeldBondsByISINs(missing)
	if len(fetched) == 0 {
		return universe
	}
	scored := s.enrichFetchedBonds(fetched, keyRate, taxRate)
	return MergeBondUniverse(universe, scored)
}
