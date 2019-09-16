package db

import (
	"testing"
)

func TestRandomBatchName(t *testing.T) {
	var nameCount = 0
	for _, list := range batchNameLists {
		if nameCount < len(list) {
			nameCount = len(list)
		}
	}

	// In any iteration "group", we should never see a duplicate
	for pass := 0; pass < 1000; pass++ {
		verifyUniqueness(t, nameCount*pass, nameCount+nameCount*pass)
	}

	// Name generation should be 100% deterministic
	var n1 = RandomBatchName(5073)
	var n2 = RandomBatchName(5073)
	if n1 != n2 {
		t.Errorf("Batch #5073 generated two different names!")
	}
}

func verifyUniqueness(t *testing.T, start, end int) {
	var names = make([]string, end-start)
	for i := start; i < end; i++ {
		names[i-start] = RandomBatchName(uint32(i))
	}

	var seen = make(map[string]bool)
	for i, name := range names {
		if seen[name] {
			t.Errorf("Already seen %q", name)
		}
		seen[name] = true
		t.Logf("Batch name %d: %q", i, name)
	}
}
