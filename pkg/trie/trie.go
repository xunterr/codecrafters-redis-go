package trie

import (
	"errors"
	"fmt"
)

type Node[T interface{}] struct {
	value T
	next  map[byte]*Node[T]
}

type Trie[T interface{}] struct {
	root *Node[T]
}

func NewTrie[T interface{}]() Trie[T] {
	return Trie[T]{
		root: &Node[T]{
			next: make(map[byte]*Node[T]),
		},
	}
}

func (t *Trie[T]) Put(key string, value T) {
	bytes := []byte(key)
	node := t.root
	for _, b := range bytes {
		n, ok := node.next[b]
		if !ok {
			n = &Node[T]{
				next: make(map[byte]*Node[T]),
			}
			node.next[b] = n
		}
		node = n
	}
	node.value = value
}

func (t Trie[T]) Get(key string) (T, error) {
	node := t.root
	for _, b := range []byte(key) {
		v, ok := node.next[b]
		if !ok {
			return *new(T), errors.New(fmt.Sprintf("Can't get value with key %s: no such key", key))
		}
		node = v
	}
	return node.value, nil
}

func (t Trie[T]) GetBestMatch(key string) (string, T, error) {
	node := t.root
	var resKey []byte

	for _, b := range []byte(key) {
		v, ok := node.next[b]
		if !ok {
			break
		}
		resKey = append(resKey, b)
		node = v
	}
	return string(resKey), node.value, nil
}
