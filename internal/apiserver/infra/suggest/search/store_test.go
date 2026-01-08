package search

import "testing"

func TestSuggestByPrefixAndPinyin(t *testing.T) {
	lines := []string{
		"张三|1|13800138000|-|5",
		"张三丰|3|18888888888|-|8",
		"李四|2|13900139000|-|3",
		"张三|1|13900000000|-|5", // duplicate ID should be removed
	}

	store := Load(lines)

	out := store.Suggest("张", 5, 6)
	if len(out) != 2 {
		t.Fatalf("expected 2 results, got %d", len(out))
	}
	if out[0].ID != 3 {
		t.Fatalf("expected first id 3, got %d", out[0].ID)
	}

	abbr := store.Suggest("zsf", 3, 6)
	if len(abbr) != 1 || abbr[0].ID != 3 {
		t.Fatalf("abbr expected id 3, got %+v", abbr)
	}

	pinyin := store.Suggest("zhang", 5, 8)
	if len(pinyin) != 2 {
		t.Fatalf("pinyin expected 2 results, got %d", len(pinyin))
	}
}

func TestSuggestNumericDedupAndSort(t *testing.T) {
	lines := []string{
		"张三|1|13900139000|-|5",
		"李四|2|13900139000|-|10",
		"王五|3|18800001111|-|1",
	}

	store := Load(lines)

	out := store.Suggest("13900139000", 5, 4)
	if len(out) != 2 {
		t.Fatalf("expected 2 results, got %d", len(out))
	}
	if out[0].ID != 2 || out[0].Weight != 10 {
		t.Fatalf("expected highest weight record first, got %+v", out[0])
	}
}
