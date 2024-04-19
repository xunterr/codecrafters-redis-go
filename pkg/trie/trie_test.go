package trie

import "testing"

func CreateTestTrie() *Trie[bool] {
	key := "hi"
	trie := NewTrie[bool]()
	trie.root.next[key[0]] = &Node[bool]{
		next: map[byte]*Node[bool]{
			key[1]: {
				value: true,
			},
		},
	}
	return trie
}

func TestTriePut(t *testing.T) {
	trie := NewTrie[bool]()
	key := "hello"
	trie.Put(key, true)

	node := trie.root
	for _, b := range []byte(key) {
		v, ok := node.next[b]
		if !ok {
			t.Fatal("Can't find next node!")
		}
		node = v
	}
	if node.value != true {
		t.Fatalf("Wrong value. Have: %v, want: %v", node.value, true)
	}
}

func TestTrieGet(t *testing.T) {
	trie := CreateTestTrie()
	val, err := trie.Get("hi")
	if err != nil || val != true {
		t.Fatal("Failed to get value")
	}
}

func TestTrieBestMatch(t *testing.T) {
	trie := CreateTestTrie()
	_, v, err := trie.GetBestMatch("hiwowo")
	if err != nil || *v != true {
		t.Fatal("Failed to get best match")
	}
}
