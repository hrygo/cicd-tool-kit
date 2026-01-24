package skill

import (
	"os"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()

	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	if r.Count() != 0 {
		t.Errorf("NewRegistry() Count = %d, want 0", r.Count())
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	skill := &Skill{
		Metadata: Metadata{
			Name:    "test-skill",
			Version: "1.0.0",
		},
	}

	err := r.Register(skill)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if !r.Exists("test-skill") {
		t.Error("Register() skill not found")
	}

	if r.Count() != 1 {
		t.Errorf("Register() Count = %d, want 1", r.Count())
	}
}

func TestRegistry_RegisterNil(t *testing.T) {
	r := NewRegistry()

	err := r.Register(nil)
	if err == nil {
		t.Error("Register(nil) expected error, got nil")
	}
}

func TestRegistry_RegisterInvalid(t *testing.T) {
	r := NewRegistry()

	skill := &Skill{
		Metadata: Metadata{
			Name: "", // Invalid
		},
	}

	err := r.Register(skill)
	if err == nil {
		t.Error("Register(invalid) expected error, got nil")
	}
}

func TestRegistry_RegisterReplace(t *testing.T) {
	r := NewRegistry()

	skill1 := &Skill{
		Metadata: Metadata{
			Name:    "test-skill",
			Version: "1.0.0",
		},
	}

	skill2 := &Skill{
		Metadata: Metadata{
			Name:    "test-skill",
			Version: "2.0.0",
		},
	}

	r.Register(skill1)
	r.Register(skill2)

	got := r.Get("test-skill")
	if got.Version != "2.0.0" {
		t.Errorf("Register() replace failed, got version %s, want 2.0.0", got.Version)
	}
}

func TestRegistry_Unregister(t *testing.T) {
	r := NewRegistry()

	skill := &Skill{
		Metadata: Metadata{
			Name:    "test-skill",
			Version: "1.0.0",
		},
	}

	r.Register(skill)
	r.Unregister("test-skill")

	if r.Exists("test-skill") {
		t.Error("Unregister() skill still exists")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()

	// Get non-existent
	got := r.Get("nonexistent")
	if got != nil {
		t.Error("Get() non-existent returned non-nil")
	}

	// Get existing
	skill := &Skill{
		Metadata: Metadata{
			Name:    "test-skill",
			Version: "1.0.0",
		},
	}
	r.Register(skill)

	got = r.Get("test-skill")
	if got == nil {
		t.Error("Get() existing returned nil")
	}

	if got.Name != "test-skill" {
		t.Errorf("Get() Name = %v, want 'test-skill'", got.Name)
	}
}

func TestRegistry_MustGet(t *testing.T) {
	r := NewRegistry()

	defer func() {
		if recover() == nil {
			t.Error("MustGet(nonexistent) did not panic")
		}
	}()

	_ = r.MustGet("nonexistent")
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()

	skills := []string{"skill-1", "skill-2", "skill-3"}
	for _, name := range skills {
		r.Register(&Skill{
			Metadata: Metadata{
				Name:    name,
				Version: "1.0.0",
			},
		})
	}

	list := r.List()
	if len(list) != 3 {
		t.Errorf("List() returned %d items, want 3", len(list))
	}
}

func TestRegistry_ListSkills(t *testing.T) {
	r := NewRegistry()

	skill1 := &Skill{
		Metadata: Metadata{
			Name:    "skill-1",
			Version: "1.0.0",
		},
	}
	skill2 := &Skill{
		Metadata: Metadata{
			Name:    "skill-2",
			Version: "1.0.0",
		},
	}

	r.Register(skill1)
	r.Register(skill2)

	skills := r.ListSkills()
	if len(skills) != 2 {
		t.Errorf("ListSkills() returned %d items, want 2", len(skills))
	}
}

func TestRegistry_Clear(t *testing.T) {
	r := NewRegistry()

	r.Register(&Skill{
		Metadata: Metadata{
			Name:    "skill-1",
			Version: "1.0.0",
		},
	})

	r.Register(&Skill{
		Metadata: Metadata{
			Name:    "skill-2",
			Version: "1.0.0",
		},
	})

	r.Clear()

	if r.Count() != 0 {
		t.Errorf("Clear() Count = %d, want 0", r.Count())
	}
}

func TestRegistry_LoadFrom(t *testing.T) {
	// Create temporary skills directory
	tmpDir := t.TempDir()

	skillDir := tmpDir + "/test-skill"
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := `---
name: test-skill
version: 1.0.0
---
Test prompt`
	if err := os.WriteFile(skillDir+"/SKILL.md", []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	r := NewRegistry()
	l := NewLoader(WithSkillDirs(tmpDir))

	errs, err := r.LoadFrom(l)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	if len(errs) > 0 {
		t.Errorf("LoadFrom() errors = %v", errs)
	}

	if !r.Exists("test-skill") {
		t.Error("LoadFrom() did not load test-skill")
	}
}

func TestRegistry_RegisterAll(t *testing.T) {
	r := NewRegistry()

	skills := []*Skill{
		{
			Metadata: Metadata{
				Name:    "skill-1",
				Version: "1.0.0",
			},
		},
		{
			Metadata: Metadata{
				Name:    "skill-2",
				Version: "1.0.0",
			},
		},
	}

	err := r.RegisterAll(skills)
	if err != nil {
		t.Fatalf("RegisterAll() error = %v", err)
	}

	if r.Count() != 2 {
		t.Errorf("RegisterAll() Count = %d, want 2", r.Count())
	}
}

func TestRegistry_RegisterAllWithError(t *testing.T) {
	r := NewRegistry()

	skills := []*Skill{
		{
			Metadata: Metadata{
				Name:    "valid-skill",
				Version: "1.0.0",
			},
		},
		{
			Metadata: Metadata{
				Name: "", // Invalid
			},
		},
	}

	err := r.RegisterAll(skills)
	if err == nil {
		t.Error("RegisterAll() expected error, got nil")
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := NewRegistry()
	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(n int) {
			skill := &Skill{
				Metadata: Metadata{
					Name:    "skill",
					Version: "1.0.0",
				},
			}
			r.Register(skill)
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			_ = r.Get("skill")
			_ = r.List()
			_ = r.Count()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Should not panic
}
