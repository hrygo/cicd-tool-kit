// Package observability tests
package observability

import (
	"testing"
)

func TestNewRBAC(t *testing.T) {
	audit, _ := NewAuditLogger("")
	rbac := NewRBAC(audit)

	if rbac == nil {
		t.Fatal("NewRBAC returned nil")
	}

	// Check default roles exist
	if len(rbac.roles) != 3 {
		t.Errorf("Expected 3 default roles, got %d", len(rbac.roles))
	}
}

func TestRegisterRole(t *testing.T) {
	audit, _ := NewAuditLogger("")
	rbac := NewRBAC(audit)

	customRole := &Role{
		Name: "custom",
		Permissions: []Permission{
			PermissionRead,
			PermissionSkillRun,
		},
	}

	err := rbac.RegisterRole(customRole)
	if err != nil {
		t.Fatalf("RegisterRole failed: %v", err)
	}

	if _, exists := rbac.roles["custom"]; !exists {
		t.Error("Custom role not registered")
	}

	// Test empty role name
	err = rbac.RegisterRole(&Role{Name: ""})
	if err == nil {
		t.Error("Expected error for empty role name")
	}
}

func TestAddUser(t *testing.T) {
	audit, _ := NewAuditLogger("")
	rbac := NewRBAC(audit)

	user := &User{
		ID:    "user1",
		Name:  "Test User",
		Email: "test@example.com",
		Roles: []string{"viewer"},
	}

	err := rbac.AddUser(user)
	if err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}

	retrieved, exists := rbac.GetUser("user1")
	if !exists {
		t.Fatal("User not found")
	}

	if retrieved.Name != "Test User" {
		t.Errorf("Expected name 'Test User', got '%s'", retrieved.Name)
	}

	// Test empty user ID
	err = rbac.AddUser(&User{ID: ""})
	if err == nil {
		t.Error("Expected error for empty user ID")
	}
}

func TestAssignRole(t *testing.T) {
	audit, _ := NewAuditLogger("")
	rbac := NewRBAC(audit)

	user := &User{
		ID:    "user1",
		Name:  "Test User",
		Email: "test@example.com",
		Roles: []string{},
	}
	rbac.AddUser(user)

	err := rbac.AssignRole("user1", "developer")
	if err != nil {
		t.Fatalf("AssignRole failed: %v", err)
	}

	roles := rbac.GetUserRoles("user1")
	if len(roles) != 1 {
		t.Errorf("Expected 1 role, got %d", len(roles))
	}

	if roles[0] != "developer" {
		t.Errorf("Expected role 'developer', got '%s'", roles[0])
	}

	// Test non-existent user
	err = rbac.AssignRole("nonexistent", "viewer")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}

	// Test non-existent role
	err = rbac.AssignRole("user1", "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent role")
	}
}

func TestRevokeRole(t *testing.T) {
	audit, _ := NewAuditLogger("")
	rbac := NewRBAC(audit)

	user := &User{
		ID:    "user1",
		Name:  "Test User",
		Email: "test@example.com",
		Roles: []string{"viewer", "developer"},
	}
	rbac.AddUser(user)

	err := rbac.RevokeRole("user1", "viewer")
	if err != nil {
		t.Fatalf("RevokeRole failed: %v", err)
	}

	roles := rbac.GetUserRoles("user1")
	if len(roles) != 1 {
		t.Errorf("Expected 1 role, got %d", len(roles))
	}

	if roles[0] != "developer" {
		t.Errorf("Expected role 'developer', got '%s'", roles[0])
	}
}

func TestHasPermission(t *testing.T) {
	audit, _ := NewAuditLogger("")
	rbac := NewRBAC(audit)

	user := &User{
		ID:    "user1",
		Name:  "Test User",
		Email: "test@example.com",
		Roles: []string{"viewer"},
	}
	rbac.AddUser(user)

	// Viewer should have read permission
	if !rbac.HasPermission("user1", PermissionRead) {
		t.Error("Viewer should have read permission")
	}

	// Viewer should not have write permission
	if rbac.HasPermission("user1", PermissionWrite) {
		t.Error("Viewer should not have write permission")
	}

	// Non-existent user should not have permission
	if rbac.HasPermission("nonexistent", PermissionRead) {
		t.Error("Non-existent user should not have permission")
	}
}

func TestHasAllPermissions(t *testing.T) {
	audit, _ := NewAuditLogger("")
	rbac := NewRBAC(audit)

	user := &User{
		ID:    "user1",
		Name:  "Test User",
		Email: "test@example.com",
		Roles: []string{"admin"},
	}
	rbac.AddUser(user)

	// Admin should have all these permissions
	perms := []Permission{PermissionRead, PermissionWrite, PermissionDelete}
	if !rbac.HasAllPermissions("user1", perms...) {
		t.Error("Admin should have all permissions")
	}

	// Admin should not have a non-existent permission
	perms = append(perms, Permission("nonexistent"))
	if rbac.HasAllPermissions("user1", perms...) {
		t.Error("Should not have non-existent permission")
	}
}

func TestHasAnyPermission(t *testing.T) {
	audit, _ := NewAuditLogger("")
	rbac := NewRBAC(audit)

	user := &User{
		ID:    "user1",
		Name:  "Test User",
		Email: "test@example.com",
		Roles: []string{"viewer"},
	}
	rbac.AddUser(user)

	// Viewer has read but not write
	perms := []Permission{PermissionRead, PermissionWrite}
	if !rbac.HasAnyPermission("user1", perms...) {
		t.Error("Viewer should have at least one of these permissions")
	}

	// Viewer has neither of these
	perms = []Permission{PermissionDelete, PermissionAdmin}
	if rbac.HasAnyPermission("user1", perms...) {
		t.Error("Viewer should not have any of these permissions")
	}
}

func TestCheckPermission(t *testing.T) {
	audit, _ := NewAuditLogger("")
	rbac := NewRBAC(audit)

	user := &User{
		ID:    "user1",
		Name:  "Test User",
		Email: "test@example.com",
		Roles: []string{"viewer"},
	}
	rbac.AddUser(user)

	// Should succeed for read permission
	err := rbac.CheckPermission("user1", PermissionRead)
	if err != nil {
		t.Errorf("CheckPermission for read failed: %v", err)
	}

	// Should fail for write permission
	err = rbac.CheckPermission("user1", PermissionWrite)
	if err == nil {
		t.Error("Expected error for write permission")
	}
}

func TestDisableUser(t *testing.T) {
	audit, _ := NewAuditLogger("")
	rbac := NewRBAC(audit)

	user := &User{
		ID:    "user1",
		Name:  "Test User",
		Email: "test@example.com",
		Roles: []string{"admin"},
	}
	rbac.AddUser(user)

	err := rbac.DisableUser("user1")
	if err != nil {
		t.Fatalf("DisableUser failed: %v", err)
	}

	// Disabled user should not have permissions
	if rbac.HasPermission("user1", PermissionRead) {
		t.Error("Disabled user should not have permissions")
	}

	// Enable user again
	err = rbac.EnableUser("user1")
	if err != nil {
		t.Fatalf("EnableUser failed: %v", err)
	}

	// Now should have permissions again
	if !rbac.HasPermission("user1", PermissionRead) {
		t.Error("Enabled user should have permissions")
	}
}

func TestListUsers(t *testing.T) {
	audit, _ := NewAuditLogger("")
	rbac := NewRBAC(audit)

	rbac.AddUser(&User{ID: "user1", Name: "User 1"})
	rbac.AddUser(&User{ID: "user2", Name: "User 2"})

	users := rbac.ListUsers()
	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}

func TestListRoles(t *testing.T) {
	audit, _ := NewAuditLogger("")
	rbac := NewRBAC(audit)

	roles := rbac.ListRoles()
	if len(roles) != 3 {
		t.Errorf("Expected 3 default roles, got %d", len(roles))
	}
}

func TestResourceGuard(t *testing.T) {
	audit, _ := NewAuditLogger("")
	rbac := NewRBAC(audit)
	guard := NewResourceGuard(rbac, audit)

	// Create resource
	err := guard.CreateResource("repo1", "repository", "owner1")
	if err != nil {
		t.Fatalf("CreateResource failed: %v", err)
	}

	// Owner should have access
	if !guard.CheckResourceAccess("repo1", "owner1", PermissionRead) {
		t.Error("Owner should have read access")
	}

	// Non-owner should not have access
	if guard.CheckResourceAccess("repo1", "user2", PermissionRead) {
		t.Error("Non-owner should not have access")
	}

	// Grant permission to another user
	err = guard.GrantResourcePermission("repo1", "user2", PermissionRead)
	if err != nil {
		t.Fatalf("GrantResourcePermission failed: %v", err)
	}

	// Now user2 should have read access
	if !guard.CheckResourceAccess("repo1", "user2", PermissionRead) {
		t.Error("user2 should have read access after grant")
	}
}

func TestPermissionMatchesWildcard(t *testing.T) {
	// Test exact match
	p := Permission("skill:run")
	if !p.MatchesWildcard(Permission("skill:run")) {
		t.Error("Exact match should return true")
	}

	// Test wildcard match
	p = Permission("skill:run")
	if !p.MatchesWildcard(Permission("skill:*")) {
		t.Error("skill:run should match skill:*")
	}

	// Test non-matching wildcard
	p = Permission("config:read")
	if p.MatchesWildcard(Permission("skill:*")) {
		t.Error("config:read should not match skill:*")
	}

	// Test non-wildcard comparison
	p = Permission("skill:run")
	if p.MatchesWildcard(Permission("skill:write")) {
		t.Error("skill:run should not match skill:write")
	}
}
