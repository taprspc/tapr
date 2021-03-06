// Copyright 2018 Klaus Birkelund Abildgaard Jensen
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package changer

import (
	"tapr.space/errors"
	"tapr.space/store/tape"
)

// Constructor is a function that creates a Changer.
type Constructor func(map[string]interface{}) (Changer, error)

var registration = make(map[string]Constructor)

// Register registers a new changer.Changer implementation.
func Register(name string, fn Constructor) error {
	const op = "changer.Register"
	if _, exists := registration[name]; exists {
		return errors.E(op, errors.Exist)
	}

	registration[name] = fn

	return nil
}

// Create creates a new Changer using the named implementation.
func Create(name string, cfg map[string]interface{}) (Changer, error) {
	const op = "changer.Create"

	fn, found := registration[name]
	if !found {
		return nil, errors.E(op, errors.Invalid, errors.Strf("unknown changer backend type: %v", name))
	}

	return fn(cfg)
}

// A Changer is an automated media changer. A changer MUST be safe for
// concurrent use.
type Changer interface {
	// Transfer moves media from src to dst.
	//
	// src and dst must be storage slots.
	Transfer(src, dst tape.Location) error

	// Load moves media from source to destination. If dst is nil, the volume
	// will be loaded into the first available data transfer slot.
	//
	// dst must be a data transfer slot.
	Load(src, dst tape.Location) error

	// Unload moves volumes from src to dst. If dst is nil, the volume will be
	// returned to its home slot (if available).
	//
	// src must be a data transfer slot.
	Unload(src, dst tape.Location) error

	// Status returns info about the changer and a list of slots.
	Status() (map[tape.SlotCategory]tape.Slots, error)
}
