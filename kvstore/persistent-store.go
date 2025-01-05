package kvstore

type PersistentStore interface {
	Set(key, value string) error
	Get(key string) (string, bool)
	Delete(key string) error
	Dump() map[string]string
	Restore(data map[string]string) error
}
