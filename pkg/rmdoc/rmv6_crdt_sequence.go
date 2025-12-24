package rmdoc

import (
	"fmt"
	"sort"

	"github.com/pkg/errors"
)

// RMV6CrdtID represents the v6 CRDT identifier/timestamp used in .rm scene files.
// It mirrors rmscene's CrdtId(part1:uint8, part2:varuint).
type RMV6CrdtID struct {
	Part1 uint8
	Part2 uint64
}

func (id RMV6CrdtID) String() string {
	return fmt.Sprintf("CrdtId(%d,%d)", id.Part1, id.Part2)
}

// RMV6EndMarker is the sentinel ID used in CRDT sequences to represent "start/end".
// This matches rmscene's END_MARKER = CrdtId(0,0).
var RMV6EndMarker = RMV6CrdtID{Part1: 0, Part2: 0}

func rmv6LessID(a, b RMV6CrdtID) bool {
	if a.Part1 != b.Part1 {
		return a.Part1 < b.Part1
	}
	return a.Part2 < b.Part2
}

// RMV6CrdtSequenceItem represents an item in a CRDT sequence, with partial ordering
// established by left/right neighbor IDs and a deleted length.
//
// This is a structural port of rmscene's CrdtSequenceItem.
type RMV6CrdtSequenceItem[T any] struct {
	ItemID        RMV6CrdtID
	LeftID        RMV6CrdtID
	RightID       RMV6CrdtID
	DeletedLength uint32
	Value         T
}

// RMV6CrdtSequence is an ordered CRDT sequence container.
// Iteration yields item IDs in deterministic order (toposort-based).
//
// This is intended as foundational infrastructure for V6 scene decoding (RMQ-0004 task 34).
type RMV6CrdtSequence[T any] struct {
	items map[RMV6CrdtID]RMV6CrdtSequenceItem[T]
}

func NewRMV6CrdtSequence[T any]() *RMV6CrdtSequence[T] {
	return &RMV6CrdtSequence[T]{items: map[RMV6CrdtID]RMV6CrdtSequenceItem[T]{}}
}

func (s *RMV6CrdtSequence[T]) Add(item RMV6CrdtSequenceItem[T]) error {
	if s.items == nil {
		s.items = map[RMV6CrdtID]RMV6CrdtSequenceItem[T]{}
	}
	if _, ok := s.items[item.ItemID]; ok {
		return errors.Errorf("crdt sequence already has item %s", item.ItemID.String())
	}
	s.items[item.ItemID] = item
	return nil
}

func (s *RMV6CrdtSequence[T]) SequenceItems() []RMV6CrdtSequenceItem[T] {
	out := make([]RMV6CrdtSequenceItem[T], 0, len(s.items))
	for _, it := range s.items {
		out = append(out, it)
	}
	return out
}

func (s *RMV6CrdtSequence[T]) Keys() ([]RMV6CrdtID, error) {
	return rmv6ToposortItems(s.items)
}

func (s *RMV6CrdtSequence[T]) Values() ([]T, error) {
	keys, err := s.Keys()
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(keys))
	for _, k := range keys {
		out = append(out, s.items[k].Value)
	}
	return out, nil
}

func (s *RMV6CrdtSequence[T]) Items() ([]RMV6CrdtSequenceItem[T], error) {
	keys, err := s.Keys()
	if err != nil {
		return nil, err
	}
	out := make([]RMV6CrdtSequenceItem[T], 0, len(keys))
	for _, k := range keys {
		out = append(out, s.items[k])
	}
	return out, nil
}

type rmv6TopoKey struct {
	Kind uint8 // 0=start, 1=item, 2=end
	ID   RMV6CrdtID
}

func rmv6ItemKey(id RMV6CrdtID) rmv6TopoKey { return rmv6TopoKey{Kind: 1, ID: id} }

var (
	rmv6StartKey = rmv6TopoKey{Kind: 0}
	rmv6EndKey   = rmv6TopoKey{Kind: 2}
)

func rmv6ToposortItems[T any](items map[RMV6CrdtID]RMV6CrdtSequenceItem[T]) ([]RMV6CrdtID, error) {
	if len(items) == 0 {
		return nil, nil
	}

	sideKey := func(sideID RMV6CrdtID, side string) rmv6TopoKey {
		// Mirrors rmscene:
		// - END_MARKER -> "__start"/"__end"
		// - unknown IDs -> "__start"/"__end" (ignored)
		if sideID == RMV6EndMarker {
			if side == "left" {
				return rmv6StartKey
			}
			return rmv6EndKey
		}
		if _, ok := items[sideID]; !ok {
			if side == "left" {
				return rmv6StartKey
			}
			return rmv6EndKey
		}
		return rmv6ItemKey(sideID)
	}

	// data is a dependency graph: key "comes after" all values in deps.
	data := map[rmv6TopoKey]map[rmv6TopoKey]struct{}{}
	addDep := func(k rmv6TopoKey, dep rmv6TopoKey) {
		m, ok := data[k]
		if !ok {
			m = map[rmv6TopoKey]struct{}{}
			data[k] = m
		}
		m[dep] = struct{}{}
	}

	for id, item := range items {
		_ = id // match item.ItemID
		left := sideKey(item.LeftID, "left")
		right := sideKey(item.RightID, "right")
		addDep(rmv6ItemKey(item.ItemID), left)
		addDep(right, rmv6ItemKey(item.ItemID))
	}

	// fill in sources not explicitly included (deps that never appeared as keys)
	deps := map[rmv6TopoKey]struct{}{}
	for _, ds := range data {
		for dep := range ds {
			deps[dep] = struct{}{}
		}
	}
	for dep := range deps {
		if _, ok := data[dep]; !ok {
			data[dep] = map[rmv6TopoKey]struct{}{}
		}
	}

	out := make([]RMV6CrdtID, 0, len(items))

	for {
		// nextItems = all nodes with no deps
		nextItems := make([]rmv6TopoKey, 0)
		for k, ds := range data {
			if len(ds) == 0 {
				nextItems = append(nextItems, k)
			}
		}

		if len(nextItems) == 1 && nextItems[0] == rmv6EndKey {
			break
		}
		if len(nextItems) == 0 {
			return nil, errors.New("cyclic dependency")
		}

		// Deterministic iteration: sort nextItems so we yield item IDs in stable order.
		sort.Slice(nextItems, func(i, j int) bool {
			a, b := nextItems[i], nextItems[j]
			if a.Kind != b.Kind {
				return a.Kind < b.Kind
			}
			return rmv6LessID(a.ID, b.ID)
		})

		// Yield item nodes only (ignore start/end markers).
		for _, k := range nextItems {
			if k.Kind == 1 {
				out = append(out, k.ID)
			}
		}

		// Remove nextItems from graph and from all dependency sets.
		nextSet := map[rmv6TopoKey]struct{}{}
		for _, k := range nextItems {
			nextSet[k] = struct{}{}
		}

		newData := map[rmv6TopoKey]map[rmv6TopoKey]struct{}{}
		for k, ds := range data {
			if _, drop := nextSet[k]; drop {
				continue
			}
			newDeps := map[rmv6TopoKey]struct{}{}
			for dep := range ds {
				if _, drop := nextSet[dep]; drop {
					continue
				}
				newDeps[dep] = struct{}{}
			}
			newData[k] = newDeps
		}
		data = newData
	}

	// Ensure stable ordering if multiple items were yielded in the same iteration,
	// and guard against any accidental duplicates (shouldn't happen, but cheap safety).
	// We keep the order already produced by the algorithm and don't re-sort out.
	if len(out) != len(items) {
		// The algorithm yields items in a graph-constrained order. If some items were
		// not yielded, it's most likely a cycle, but allow a clearer error.
		return nil, errors.Errorf("toposort yielded %d items, expected %d", len(out), len(items))
	}

	// Verify uniqueness
	seen := map[RMV6CrdtID]struct{}{}
	for _, id := range out {
		if _, ok := seen[id]; ok {
			return nil, errors.Errorf("toposort produced duplicate id %s", id.String())
		}
		seen[id] = struct{}{}
	}

	return out, nil
}
