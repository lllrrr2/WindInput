package dict

import (
	"sort"

	"github.com/huanfeng/wind_input/internal/candidate"
)

// TrieNode Trie 树节点
type TrieNode struct {
	children   map[byte]*TrieNode
	candidates []candidate.Candidate
	isEnd      bool
}

// Trie 前缀树
type Trie struct {
	root       *TrieNode
	entryCount int
}

// NewTrie 创建空的 Trie 树
func NewTrie() *Trie {
	return &Trie{
		root: &TrieNode{
			children: make(map[byte]*TrieNode),
		},
	}
}

// Insert 插入一个词条
func (t *Trie) Insert(code string, cand candidate.Candidate) {
	node := t.root
	for i := 0; i < len(code); i++ {
		c := code[i]
		if node.children == nil {
			node.children = make(map[byte]*TrieNode)
		}
		child, ok := node.children[c]
		if !ok {
			child = &TrieNode{
				children: make(map[byte]*TrieNode),
			}
			node.children[c] = child
		}
		node = child
	}
	node.isEnd = true
	node.candidates = append(node.candidates, cand)
	t.entryCount++
}

// Search 精确查找
func (t *Trie) Search(code string) []candidate.Candidate {
	node := t.findNode(code)
	if node == nil || !node.isEnd {
		return nil
	}
	return node.candidates
}

// SearchPrefix 前缀查找，返回所有以 prefix 为前缀的词条
func (t *Trie) SearchPrefix(prefix string, limit int) []candidate.Candidate {
	node := t.findNode(prefix)
	if node == nil {
		return nil
	}

	var results []candidate.Candidate
	t.collectAll(node, &results, limit)

	sort.Slice(results, func(i, j int) bool {
		return results[i].Weight > results[j].Weight
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// HasPrefix 检查是否有以 prefix 开头的词条
func (t *Trie) HasPrefix(prefix string) bool {
	return t.findNode(prefix) != nil
}

// EntryCount 返回总词条数
func (t *Trie) EntryCount() int {
	return t.entryCount
}

// findNode 沿 code 路径查找节点
func (t *Trie) findNode(code string) *TrieNode {
	node := t.root
	for i := 0; i < len(code); i++ {
		child, ok := node.children[code[i]]
		if !ok {
			return nil
		}
		node = child
	}
	return node
}

// collectAll 收集节点及其所有子节点的候选词
func (t *Trie) collectAll(node *TrieNode, results *[]candidate.Candidate, limit int) {
	if limit > 0 && len(*results) >= limit {
		return
	}
	if node.isEnd {
		*results = append(*results, node.candidates...)
	}
	for _, child := range node.children {
		if limit > 0 && len(*results) >= limit {
			return
		}
		t.collectAll(child, results, limit)
	}
}
