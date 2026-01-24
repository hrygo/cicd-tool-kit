package skill

import (
	"fmt"
	"sync"
)

// Registry manages a collection of available skills.
type Registry struct {
	mu     sync.RWMutex
	skills map[string]*Skill // name -> Skill
}

// NewRegistry creates a new empty skill registry.
func NewRegistry() *Registry {
	return &Registry{
		skills: make(map[string]*Skill),
	}
}

// Register adds a skill to the registry.
// If a skill with the same name exists, it is replaced.
func (r *Registry) Register(skill *Skill) error {
	if skill == nil {
		return fmt.Errorf("cannot register nil skill")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate before registering
	if err := skill.Validate(); err != nil {
		return fmt.Errorf("skill validation failed: %w", err)
	}

	r.skills[skill.Name] = skill
	return nil
}

// Unregister removes a skill from the registry.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.skills, name)
}

// Get retrieves a skill by name.
// Returns nil if not found.
func (r *Registry) Get(name string) *Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.skills[name]
}

// MustGet retrieves a skill by name or panics if not found.
func (r *Registry) MustGet(name string) *Skill {
	skill := r.Get(name)
	if skill == nil {
		panic(fmt.Sprintf("skill not found: %s", name))
	}
	return skill
}

// List returns all registered skill names.
// Order is not guaranteed due to map iteration randomness.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.skills))
	// Pre-allocated capacity makes append efficient
	for name := range r.skills {
		names = append(names, name)
	}
	return names
}

// ListSkills returns all registered skills.
// Order is not guaranteed due to map iteration randomness.
func (r *Registry) ListSkills() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills := make([]*Skill, 0, len(r.skills))
	// Pre-allocated capacity makes append efficient
	for _, skill := range r.skills {
		skills = append(skills, skill)
	}
	return skills
}

// Count returns the number of registered skills.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.skills)
}

// Exists checks if a skill is registered.
func (r *Registry) Exists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.skills[name]
	return ok
}

// Clear removes all skills from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.skills = make(map[string]*Skill)
}

// LoadFrom loads skills from a loader into the registry.
func (r *Registry) LoadFrom(loader *Loader) ([]error, error) {
	skills, errs := loader.Discover()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Add all discovered skills
	for name, skill := range skills {
		r.skills[name] = skill
	}

	return errs, nil
}

// RegisterAll registers multiple skills at once.
// Stops at first error and returns it.
func (r *Registry) RegisterAll(skills []*Skill) error {
	for _, skill := range skills {
		if err := r.Register(skill); err != nil {
			return fmt.Errorf("failed to register %s: %w", skill.Name, err)
		}
	}
	return nil
}
