package kvstore

type PersistentStore interface {
	Set(key, value string) error
	Get(key string) (string, bool)
}
