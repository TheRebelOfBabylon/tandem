package storage

type StorageBackend interface {
	Close() error
}
