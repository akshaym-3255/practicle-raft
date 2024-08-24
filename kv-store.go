package main

type keyValueStore struct {
	store map[string]string
}

func newKeyValueStore() *keyValueStore {
	return &keyValueStore{store: make(map[string]string)}
}

func (kv *keyValueStore) set(key, value string) {
	kv.store[key] = value
}

func (kv *keyValueStore) get(key string) (string, bool) {
	value, ok := kv.store[key]
	return value, ok
}
