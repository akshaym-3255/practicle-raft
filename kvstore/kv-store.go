package kvstore

import (
	"github.com/akshaym-3255/practicle-raft/logger"
)

type KeyValueStore struct {
	PersistentStore PersistentStore
}

func NewKeyValueStore(store PersistentStore) *KeyValueStore {
	return &KeyValueStore{store}
}

func (kv *KeyValueStore) Set(key, value string) {
	err := kv.PersistentStore.Set(key, value)
	if err != nil {
		logger.Log.Errorln("Unable to add key value please check ")
	}
}

func (kv *KeyValueStore) Get(key string) (string, bool) {
	return kv.PersistentStore.Get(key)
}

func (kv *KeyValueStore) Delete(key string) {
	err := kv.PersistentStore.Delete(key)
	if err != nil {
		logger.Log.Errorln("Unable to delete key please check ")
	}
}

func (kv *KeyValueStore) Dump() map[string]string {
	return kv.PersistentStore.Dump()
}

func (kv *KeyValueStore) Restore(data map[string]string) error {
	return kv.PersistentStore.Restore(data)
}
