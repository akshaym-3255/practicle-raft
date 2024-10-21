package kvstore

type PersistentStore interface {
	Set(key, value string) error
	Get(key string) (string, bool)
	Dump() map[string]string
	Restore(data map[string]string) error
}
