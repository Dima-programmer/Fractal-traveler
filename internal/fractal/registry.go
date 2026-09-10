package fractal

import "sync"

// Registry maps fractal keys to factories and holds the built-in types.
type Registry struct {
	mu    sync.RWMutex
	types map[string]func() Fractal
}

// NewRegistry returns a registry pre-populated with all built-in fractals.
func NewRegistry() *Registry {
	r := &Registry{types: make(map[string]func() Fractal)}
	r.Register(func() Fractal { return NewMandelbrot() })
	r.Register(func() Fractal { return NewJulia() })
	r.Register(func() Fractal { return NewBurningShip() })
	r.Register(func() Fractal { return NewNewton() })
	r.Register(func() Fractal { return NewTricorn() })
	r.Register(func() Fractal { return NewPhoenix() })
	return r
}

// Register adds a fractal factory under its Key.
// It overwrites any existing entry with the same key.
func (r *Registry) Register(factory func() Fractal) {
	f := factory()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.types[f.Key()] = factory
}

// Keys returns all registered fractal keys in stable order.
func (r *Registry) Keys() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]string, 0, len(r.types))
	for k := range r.types {
		keys = append(keys, k)
	}
	return keys
}

// IsRegistered reports whether a key exists.
func (r *Registry) IsRegistered(key string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.types[key]
	return ok
}

// Create instantiates a new fractal of the given key.
// Returns nil if the key is unknown.
func (r *Registry) Create(key string) Fractal {
	r.mu.RLock()
	factory, ok := r.types[key]
	r.mu.RUnlock()
	if !ok {
		return nil
	}
	return factory()
}
