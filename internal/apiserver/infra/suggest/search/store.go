package search

import (
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/suggest"
)

type Store struct {
	trie  *Trie
	table *Hash
	mu    sync.RWMutex
}

var active atomic.Value // holds *Store

// Load builds a Store from raw lines.
func Load(lines []string) *Store {
	t := NewTrie()
	h := NewHash()
	t.ImportLines(lines)
	h.ImportLines(lines)
	return &Store{trie: t, table: h}
}

// Swap atomically replaces the active store.
func Swap(s *Store) { active.Store(s) }

// Current returns the current store.
func Current() *Store {
	if v := active.Load(); v != nil {
		return v.(*Store)
	}
	return nil
}

// ImportLines appends new data into the existing store (protected by lock).
func (s *Store) ImportLines(lines []string) {
	if s == nil || len(lines) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.trie.ImportLines(lines)
	s.table.ImportLines(lines)
}

// Suggest returns ordered and deduplicated terms.
func (s *Store) Suggest(keyword string, max int, pad int) []suggest.Term {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if max <= 0 {
		max = 20
	}

	// 数字走 Hash
	if isDigits(keyword) {
		terms := Terms(s.table.Search(keyword))
		terms = RemoveDuplicate(terms)
		sort.Sort(terms)
		if len(terms) > max {
			terms = terms[:max]
		}
		return terms
	}
	// 前缀通配
	k := keyword
	if len([]rune(k)) < pad {
		k = k + strings.Repeat("*", pad-len([]rune(k)))
	}
	keys := s.trie.Wildcard(k)
	var out Terms
	for _, key := range keys {
		if v := s.trie.Get(key); v != nil {
			out = append(out, v.(Terms)...)
		}
	}
	out = RemoveDuplicate(out)
	sort.Sort(out)
	if len(out) > max {
		out = out[:max]
	}
	return out
}

func isDigits(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return len(s) > 0
}
