package kvstore

import "fmt"

type KeyValueStore struct {
	PersistentStore PersistentStore
}

func NewKeyValueStore(store PersistentStore) *KeyValueStore {
	return &KeyValueStore{store}
}

func (kv *KeyValueStore) Set(key, value string) {

	fmt.Println("Setting key value pair in kvstore")
	err := kv.PersistentStore.Set(key, value)
	if err != nil {
		return
	}
}

func (kv *KeyValueStore) Get(key string) (string, bool) {
	return kv.PersistentStore.Get(key)
}
