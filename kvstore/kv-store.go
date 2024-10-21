package kvstore

type KeyValueStore struct {
	PersistentStore PersistentStore
}

func NewKeyValueStore(store PersistentStore) *KeyValueStore {
	return &KeyValueStore{store}
}

func (kv *KeyValueStore) Set(key, value string) {
	err := kv.PersistentStore.Set(key, value)
	if err != nil {
		return
	}
}

func (kv *KeyValueStore) Get(key string) (string, bool) {
	return kv.PersistentStore.Get(key)
}

func (kv *KeyValueStore) Dump() map[string]string {
	return kv.PersistentStore.Dump()
}

func (kv *KeyValueStore) Restore(data map[string]string) error {
	return kv.PersistentStore.Restore(data)
}
