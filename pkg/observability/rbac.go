// Package observability provides RBAC (Role-Based Access Control) functionality
package observability

import (
	"fmt"
	"strings"
	"sync"
)

// Permission represents a specific action that can be performed
type Permission string

const (
	// Resource permissions
	PermissionRead   Permission = "read"
	PermissionWrite  Permission = "write"
	PermissionDelete Permission = "delete"
	PermissionAdmin  Permission = "admin"

	// Skill permissions
	PermissionSkillRun   Permission = "skill:run"
	PermissionSkillCreate Permission = "skill:create"
	PermissionSkillModify Permission = "skill:modify"
	PermissionSkillDelete Permission = "skill:delete"

	// Config permissions
	PermissionConfigRead  Permission = "config:read"
	PermissionConfigWrite Permission = "config:write"

	// Audit permissions
	PermissionAuditRead Permission = "audit:read"
	PermissionAuditWrite Permission = "audit:write"
)

// Role represents a user role with associated permissions
type Role struct {
	Name        string
	Permissions []Permission
}

// Default roles
var (
	RoleViewer = Role{
		Name: "viewer",
		Permissions: []Permission{
			PermissionRead,
			PermissionSkillRun,
			PermissionConfigRead,
			PermissionAuditRead,
		},
	}

	RoleDeveloper = Role{
		Name: "developer",
		Permissions: []Permission{
			PermissionRead,
			PermissionWrite,
			PermissionSkillRun,
			PermissionSkillCreate,
			PermissionSkillModify,
			PermissionConfigRead,
			PermissionAuditRead,
		},
	}

	RoleAdmin = Role{
		Name: "admin",
		Permissions: []Permission{
			PermissionRead,
			PermissionWrite,
			PermissionDelete,
			PermissionAdmin,
			PermissionSkillRun,
			PermissionSkillCreate,
			PermissionSkillModify,
			PermissionSkillDelete,
			PermissionConfigRead,
			PermissionConfigWrite,
			PermissionAuditRead,
			PermissionAuditWrite,
		},
	}
)

// User represents a system user with roles
type User struct {
	ID       string
	Name     string
	Email    string
	Roles    []string
	Disabled bool
}

// RBAC provides role-based access control
type RBAC struct {
	mu        sync.RWMutex
	roles     map[string]*Role
	users     map[string]*User
	userRoles map[string][]string // user ID -> role names
	audit     *AuditLogger
}

// NewRBAC creates a new RBAC instance
func NewRBAC(audit *AuditLogger) *RBAC {
	rbac := &RBAC{
		roles:     make(map[string]*Role),
		users:     make(map[string]*User),
		userRoles: make(map[string][]string),
		audit:     audit,
	}

	// Register default roles
	rbac.RegisterRole(&RoleViewer)
	rbac.RegisterRole(&RoleDeveloper)
	rbac.RegisterRole(&RoleAdmin)

	return rbac
}

// RegisterRole registers a new role
func (r *RBAC) RegisterRole(role *Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if role.Name == "" {
		return fmt.Errorf("role name cannot be empty")
	}

	r.roles[role.Name] = role
	return nil
}

// AddUser adds a new user
func (r *RBAC) AddUser(user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if user.ID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	r.users[user.ID] = user
	r.userRoles[user.ID] = user.Roles

	if r.audit != nil {
		r.audit.LogEvent("info", "user_added", "create_user", map[string]interface{}{
			"user_id":   user.ID,
			"user_name": user.Name,
			"user_email": user.Email,
			"roles":     user.Roles,
		})
	}

	return nil
}

// AssignRole assigns a role to a user
func (r *RBAC) AssignRole(userID, roleName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.roles[roleName]; !exists {
		return fmt.Errorf("role %s does not exist", roleName)
	}

	if _, exists := r.users[userID]; !exists {
		return fmt.Errorf("user %s does not exist", userID)
	}

	// Check if user already has this role
	for _, role := range r.userRoles[userID] {
		if role == roleName {
			return nil // Already has the role
		}
	}

	r.userRoles[userID] = append(r.userRoles[userID], roleName)

	if r.audit != nil {
		r.audit.LogEvent("info", "role_assigned", "assign_role", map[string]interface{}{
			"user_id":   userID,
			"role_name": roleName,
		})
	}

	return nil
}

// RevokeRole revokes a role from a user
func (r *RBAC) RevokeRole(userID, roleName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	roles := r.userRoles[userID]
	newRoles := make([]string, 0, len(roles))
	found := false

	for _, role := range roles {
		if role != roleName {
			newRoles = append(newRoles, role)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("user %s does not have role %s", userID, roleName)
	}

	r.userRoles[userID] = newRoles

	if r.audit != nil {
		r.audit.LogEvent("info", "role_revoked", "revoke_role", map[string]interface{}{
			"user_id":   userID,
			"role_name": roleName,
		})
	}

	return nil
}

// HasPermission checks if a user has a specific permission
func (r *RBAC) HasPermission(userID string, permission Permission) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if user exists and is not disabled
	user, exists := r.users[userID]
	if !exists || user.Disabled {
		return false
	}

	// Check all user roles for the permission
	for _, roleName := range r.userRoles[userID] {
		role := r.roles[roleName]
		if role != nil {
			for _, p := range role.Permissions {
				if p == permission {
					return true
				}
			}
		}
	}

	return false
}

// HasAnyPermission checks if a user has any of the specified permissions
func (r *RBAC) HasAnyPermission(userID string, permissions ...Permission) bool {
	for _, perm := range permissions {
		if r.HasPermission(userID, perm) {
			return true
		}
	}
	return false
}

// HasAllPermissions checks if a user has all of the specified permissions
func (r *RBAC) HasAllPermissions(userID string, permissions ...Permission) bool {
	for _, perm := range permissions {
		if !r.HasPermission(userID, perm) {
			return false
		}
	}
	return true
}

// CheckPermission checks permission and returns an error if not authorized
func (r *RBAC) CheckPermission(userID string, permission Permission) error {
	if !r.HasPermission(userID, permission) {
		if r.audit != nil {
			r.audit.LogAuthEvent("permission_denied", userID, string(permission), false)
		}
		return fmt.Errorf("user %s does not have permission %s", userID, permission)
	}

	if r.audit != nil {
		r.audit.LogAuthEvent("permission_granted", userID, string(permission), true)
	}

	return nil
}

// GetUser returns a user by ID
func (r *RBAC) GetUser(userID string) (*User, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[userID]
	return user, exists
}

// GetUserRoles returns all roles for a user
func (r *RBAC) GetUserRoles(userID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	roles := r.userRoles[userID]
	result := make([]string, len(roles))
	copy(result, roles)
	return result
}

// DisableUser disables a user account
func (r *RBAC) DisableUser(userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[userID]
	if !exists {
		return fmt.Errorf("user %s does not exist", userID)
	}

	user.Disabled = true

	if r.audit != nil {
		r.audit.LogEvent("warning", "user_disabled", "disable_user", map[string]interface{}{
			"user_id":   userID,
			"user_name": user.Name,
		})
	}

	return nil
}

// EnableUser enables a user account
func (r *RBAC) EnableUser(userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[userID]
	if !exists {
		return fmt.Errorf("user %s does not exist", userID)
	}

	user.Disabled = false

	if r.audit != nil {
		r.audit.LogEvent("info", "user_enabled", "enable_user", map[string]interface{}{
			"user_id":   userID,
			"user_name": user.Name,
		})
	}

	return nil
}

// ListUsers returns all users
func (r *RBAC) ListUsers() []*User {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]*User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}
	return users
}

// ListRoles returns all registered roles
func (r *RBAC) ListRoles() []*Role {
	r.mu.RLock()
	defer r.mu.RUnlock()

	roles := make([]*Role, 0, len(r.roles))
	for _, role := range r.roles {
		roles = append(roles, role)
	}
	return roles
}

// PermissionFromString parses a permission from a string
func PermissionFromString(s string) Permission {
	// Handle skill:* wildcard
	if s == "skill:*" {
		return Permission("skill:*")
	}
	// Handle config:* wildcard
	if s == "config:*" {
		return Permission("config:*")
	}
	// Handle audit:* wildcard
	if s == "audit:*" {
		return Permission("audit:*")
	}

	return Permission(s)
}

// MatchesWildcard checks if a permission matches a wildcard pattern
func (p Permission) MatchesWildcard(pattern Permission) bool {
	patternStr := string(pattern)
	permStr := string(p)

	if !strings.HasSuffix(patternStr, "*") {
		return p == pattern
	}

	prefix := strings.TrimSuffix(patternStr, "*")
	return strings.HasPrefix(permStr, prefix)
}

// Resource represents a protected resource
type Resource struct {
	Name        string
	Type        string
	OwnerUserID string
	Permissions map[string][]Permission // user ID -> permissions
}

// ResourceGuard manages per-resource permissions
type ResourceGuard struct {
	mu        sync.RWMutex
	resources map[string]*Resource
	audit     *AuditLogger
	rbac      *RBAC
}

// NewResourceGuard creates a new resource guard
func NewResourceGuard(rbac *RBAC, audit *AuditLogger) *ResourceGuard {
	return &ResourceGuard{
		resources: make(map[string]*Resource),
		audit:     audit,
		rbac:      rbac,
	}
}

// CreateResource creates a new protected resource
func (g *ResourceGuard) CreateResource(name, resourceType, ownerID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.resources[name]; exists {
		return fmt.Errorf("resource %s already exists", name)
	}

	g.resources[name] = &Resource{
		Name:        name,
		Type:        resourceType,
		OwnerUserID: ownerID,
		Permissions: make(map[string][]Permission),
	}

	// Owner gets all permissions
	g.resources[name].Permissions[ownerID] = []Permission{
		PermissionRead,
		PermissionWrite,
		PermissionDelete,
		PermissionAdmin,
	}

	return nil
}

// GrantResourcePermission grants a user permission on a resource
func (g *ResourceGuard) GrantResourcePermission(resourceName, userID string, permission Permission) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	resource, exists := g.resources[resourceName]
	if !exists {
		return fmt.Errorf("resource %s does not exist", resourceName)
	}

	// Check if grantor has admin permission on the resource
	// (In a real system, you'd pass the grantor's user ID)

	perms := resource.Permissions[userID]
	for _, p := range perms {
		if p == permission {
			return nil // Already has permission
		}
	}

	resource.Permissions[userID] = append(perms, permission)

	if g.audit != nil {
		g.audit.LogEvent("info", "resource_permission_granted", "grant_permission", map[string]interface{}{
			"resource": resourceName,
			"user_id":  userID,
			"permission": permission,
		})
	}

	return nil
}

// CheckResourceAccess checks if a user can access a resource with a specific permission
func (g *ResourceGuard) CheckResourceAccess(resourceName, userID string, required Permission) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	resource, exists := g.resources[resourceName]
	if !exists {
		return false
	}

	// Owner has all permissions
	if resource.OwnerUserID == userID {
		return true
	}

	// Check specific permissions
	for _, permission := range resource.Permissions[userID] {
		if permission == required || permission == PermissionAdmin {
			return true
		}
	}

	return false
}
