// Package store defines the interfaces that implements the basic
// 'store' abstraction.
package store // import "hpt.space/tapr/store"

import (
	"hpt.space/tapr/config"
	"hpt.space/tapr/errors"
	"hpt.space/tapr/storage"
)

// A Constructor is a function that creates a Store.
type Constructor func(name string, cfg config.StoreConfig) (Store, error)

var registration = make(map[string]Constructor)

// Register registers a new Store Constructor with the given name.
func Register(name string, fn Constructor) error {
	const op = "store.Register"
	if _, exists := registration[name]; exists {
		return errors.E(op, errors.Exist)
	}

	registration[name] = fn

	return nil
}

// Store is the store interface.
type Store interface {
	// String returns the target name of the store.
	String() string

	// embed the storage.Storage interface.
	storage.Storage
}

// Config is a store configuration. It is implementation dependent.
type Config interface{}

// Create creates a new store using the given named implementation.
func Create(name, backend string, cfg config.StoreConfig) (Store, error) {
	const op = "store.Create"

	fn, found := registration[backend]
	if !found {
		return nil, errors.E(op, errors.Invalid, errors.Strf("unknown store backend type: %v", backend))
	}

	return fn(name, cfg)
}