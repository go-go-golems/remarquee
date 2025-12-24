package rmdoc

import "testing"

func TestRMV6CrdtSequence_Toposort_DeterministicIndependent(t *testing.T) {
	seq := NewRMV6CrdtSequence[int]()

	// Two independent items: both reference start/end markers -> ordering should be deterministic by ID.
	a := RMV6CrdtSequenceItem[int]{
		ItemID:        RMV6CrdtID{Part1: 1, Part2: 10},
		LeftID:        RMV6EndMarker,
		RightID:       RMV6EndMarker,
		DeletedLength: 0,
		Value:         10,
	}
	b := RMV6CrdtSequenceItem[int]{
		ItemID:        RMV6CrdtID{Part1: 1, Part2: 5},
		LeftID:        RMV6EndMarker,
		RightID:       RMV6EndMarker,
		DeletedLength: 0,
		Value:         5,
	}

	if err := seq.Add(a); err != nil {
		t.Fatalf("Add(a): %v", err)
	}
	if err := seq.Add(b); err != nil {
		t.Fatalf("Add(b): %v", err)
	}

	keys, err := seq.Keys()
	if err != nil {
		t.Fatalf("Keys(): %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("len(keys)=%d, want 2", len(keys))
	}
	// b (part2=5) should come before a (part2=10)
	if keys[0] != b.ItemID || keys[1] != a.ItemID {
		t.Fatalf("unexpected order: got %v, want [%v %v]", keys, b.ItemID, a.ItemID)
	}
}

func TestRMV6CrdtSequence_Toposort_ChainByLeft(t *testing.T) {
	seq := NewRMV6CrdtSequence[string]()

	aID := RMV6CrdtID{Part1: 1, Part2: 1}
	bID := RMV6CrdtID{Part1: 1, Part2: 2}
	cID := RMV6CrdtID{Part1: 1, Part2: 3}

	// Build a -> b -> c by setting each item's LeftID to predecessor.
	if err := seq.Add(RMV6CrdtSequenceItem[string]{ItemID: aID, LeftID: RMV6EndMarker, RightID: RMV6EndMarker, Value: "a"}); err != nil {
		t.Fatalf("Add(a): %v", err)
	}
	if err := seq.Add(RMV6CrdtSequenceItem[string]{ItemID: bID, LeftID: aID, RightID: RMV6EndMarker, Value: "b"}); err != nil {
		t.Fatalf("Add(b): %v", err)
	}
	if err := seq.Add(RMV6CrdtSequenceItem[string]{ItemID: cID, LeftID: bID, RightID: RMV6EndMarker, Value: "c"}); err != nil {
		t.Fatalf("Add(c): %v", err)
	}

	keys, err := seq.Keys()
	if err != nil {
		t.Fatalf("Keys(): %v", err)
	}
	if len(keys) != 3 {
		t.Fatalf("len(keys)=%d, want 3", len(keys))
	}
	if keys[0] != aID || keys[1] != bID || keys[2] != cID {
		t.Fatalf("unexpected order: got %v, want [%v %v %v]", keys, aID, bID, cID)
	}
}

func TestRMV6CrdtSequence_Toposort_UnknownLeftTreatedAsStart(t *testing.T) {
	seq := NewRMV6CrdtSequence[int]()

	unknown := RMV6CrdtID{Part1: 9, Part2: 999}
	aID := RMV6CrdtID{Part1: 1, Part2: 1}
	bID := RMV6CrdtID{Part1: 1, Part2: 2}

	// b references unknown left -> treated as start, so b becomes independent.
	if err := seq.Add(RMV6CrdtSequenceItem[int]{ItemID: aID, LeftID: RMV6EndMarker, RightID: RMV6EndMarker, Value: 1}); err != nil {
		t.Fatalf("Add(a): %v", err)
	}
	if err := seq.Add(RMV6CrdtSequenceItem[int]{ItemID: bID, LeftID: unknown, RightID: RMV6EndMarker, Value: 2}); err != nil {
		t.Fatalf("Add(b): %v", err)
	}

	keys, err := seq.Keys()
	if err != nil {
		t.Fatalf("Keys(): %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("len(keys)=%d, want 2", len(keys))
	}

	// Both items are independent; ordering is deterministic by ID.
	if keys[0] != aID || keys[1] != bID {
		t.Fatalf("unexpected order: got %v, want [%v %v]", keys, aID, bID)
	}
}

func TestRMV6CrdtSequence_Toposort_Cycle(t *testing.T) {
	seq := NewRMV6CrdtSequence[int]()

	aID := RMV6CrdtID{Part1: 1, Part2: 1}
	bID := RMV6CrdtID{Part1: 1, Part2: 2}

	// Cycle by left references:
	// a depends on b, b depends on a.
	if err := seq.Add(RMV6CrdtSequenceItem[int]{ItemID: aID, LeftID: bID, RightID: RMV6EndMarker, Value: 1}); err != nil {
		t.Fatalf("Add(a): %v", err)
	}
	if err := seq.Add(RMV6CrdtSequenceItem[int]{ItemID: bID, LeftID: aID, RightID: RMV6EndMarker, Value: 2}); err != nil {
		t.Fatalf("Add(b): %v", err)
	}

	_, err := seq.Keys()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
