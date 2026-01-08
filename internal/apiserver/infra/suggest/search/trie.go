package search

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/mozillazg/go-pinyin"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/suggest"
)

const maxSearchLen = 100

// Trie implements a ternary search tree for prefix/wildcard lookup.
type Trie struct {
	root *node
}

type node struct {
	small *node
	equal *node
	large *node
	value Terms
	r     rune
	end   bool
}

type Terms []suggest.Term

func (t Terms) Len() int           { return len(t) }
func (t Terms) Less(i, j int) bool { return t[i].Weight > t[j].Weight }
func (t Terms) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }

// RemoveDuplicate deduplicates by ID while preserving order.
func RemoveDuplicate(list Terms) Terms {
	var out Terms
	for _, cur := range list {
		found := false
		for _, v := range out {
			if v.ID == cur.ID {
				found = true
				break
			}
		}
		if !found {
			out = append(out, cur)
		}
	}
	return out
}

// NewTrie creates a new Trie.
func NewTrie() *Trie {
	return &Trie{}
}

// ImportLines parses name|id|mobiles|disease|weight lines and inserts terms.
func (t *Trie) ImportLines(lines []string) {
	pyArgs := pinyin.NewArgs()
	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		if name == "" {
			continue
		}
		idStr := strings.TrimSpace(parts[1])
		id, _ := strconv.ParseInt(idStr, 10, 64)
		mobile := strings.TrimSpace(parts[2])
		_ = strings.TrimSpace(parts[3]) // disease field ignored
		weight, _ := strconv.Atoi(strings.TrimSpace(parts[4]))
		term := suggest.Term{Name: name, ID: id, Mobile: mobile, Weight: weight}

		// 原始中文名
		t.Put(name, term)
		// 拼音/简拼
		py := pinyin.Pinyin(name, pyArgs)
		if len(py) == 0 {
			continue
		}
		py[0] = uniq(py[0])
		for _, a := range py[0] {
			full, abbr := a, string(a[0])
			for _, b := range py[1:] {
				full += b[0]
				abbr += string(b[0][0])
			}
			t.Put(full, term)
			t.Put(abbr, term)
		}
	}
}

func uniq(list []string) []string {
	var out []string
	for _, s := range list {
		exists := false
		for _, v := range out {
			if s == v {
				exists = true
				break
			}
		}
		if !exists {
			out = append(out, s)
		}
	}
	return out
}

// Import 从文件导入
func (t *Trie) Import(file string) {
	f, err := os.Open(file)
	if err != nil {
		return
	}
	defer f.Close()
	r := bufio.NewReaderSize(f, 400)
	line, _, err := r.ReadLine()
	for err == nil {
		t.ImportLines([]string{string(line)})
		line, _, err = r.ReadLine()
	}
}

// Put inserts a term keyed by the provided string.
func (t *Trie) Put(key string, term suggest.Term) {
	if key == "" {
		return
	}
	t.root = t.putRecursive(t.root, []rune(key), 0, term)
}

func (t *Trie) putRecursive(n *node, key []rune, idx int, term suggest.Term) *node {
	r := key[idx]
	if n == nil {
		n = &node{r: r}
	}
	if r < n.r {
		n.small = t.putRecursive(n.small, key, idx, term)
	} else if r > n.r {
		n.large = t.putRecursive(n.large, key, idx, term)
	} else if idx < len(key)-1 {
		n.equal = t.putRecursive(n.equal, key, idx+1, term)
	} else {
		n.end = true
		n.value = append(n.value, term)
	}
	return n
}

// Get retrieves the terms for the exact key.
func (t *Trie) Get(key string) interface{} {
	n := t.root
	rkey := []rune(key)
	for i, r := range rkey {
		for n != nil {
			if r < n.r {
				n = n.small
			} else if r > n.r {
				n = n.large
			} else {
				if i == len(rkey)-1 && n.end {
					return n.value
				}
				n = n.equal
				break
			}
		}
		if n == nil {
			return nil
		}
	}
	return nil
}

// Wildcard supports '*' or '.' wildcards for prefix matching.
func (t *Trie) Wildcard(key string) []string {
	if key == "" {
		return nil
	}
	realLen := len([]rune(strings.TrimRight(key, "*")))
	return t.wildcardRecursive(t.root, []rune(key), realLen, 0, "")
}

func (t *Trie) wildcardRecursive(n *node, key []rune, realLen, idx int, prefix string) (matches []string) {
	if n == nil {
		return
	}
	if idx == len(key) {
		t.collectAll(n, prefix, &matches)
		return
	}
	r := key[idx]
	isWild := r == '*' || r == '.'
	if (isWild || r < n.r) && len(matches) < maxSearchLen {
		matches = append(matches, t.wildcardRecursive(n.small, key, realLen, idx, prefix)...)
	}
	if (isWild || r > n.r) && len(matches) < maxSearchLen {
		matches = append(matches, t.wildcardRecursive(n.large, key, realLen, idx, prefix)...)
	}
	if (isWild || r == n.r) && len(matches) < maxSearchLen {
		newPrefix := prefix + string(n.r)
		if n.end && idx >= realLen-1 {
			matches = append(matches, newPrefix)
		}
		matches = append(matches, t.wildcardRecursive(n.equal, key, realLen, idx+1, newPrefix)...)
	}
	return
}

// collectAll collects all terminal keys under the given node up to maxSearchLen.
func (t *Trie) collectAll(n *node, prefix string, matches *[]string) {
	if n == nil || len(*matches) >= maxSearchLen {
		return
	}
	// explore smaller branch without adding current rune
	t.collectAll(n.small, prefix, matches)

	cur := prefix + string(n.r)
	if n.end {
		*matches = append(*matches, cur)
		if len(*matches) >= maxSearchLen {
			return
		}
	}

	t.collectAll(n.equal, cur, matches)
	t.collectAll(n.large, prefix, matches)
}
