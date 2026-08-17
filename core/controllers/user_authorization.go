package controllers

import (
	"github.com/goastian/astiango-hub/core/constants"
	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/juju/errors"
)

// isRootAdmin is deliberately limited to the platform root administrator. A
// tenant administrator must remain inside its own tenant boundary.
func isRootAdmin(user *models.User) bool {
	return user != nil && user.RootAdmin
}

func isTenantAdmin(user *models.User) bool {
	return user != nil && user.Role == constants.RoleAdmin && !user.TenantId.IsZero()
}

func canManageUser(actor, target *models.User) bool {
	if actor == nil || target == nil {
		return false
	}
	if actor.Id == target.Id || isRootAdmin(actor) {
		return true
	}
	return isTenantAdmin(actor) && actor.TenantId == target.TenantId
}

func requireUserManagementAccess(actor, target *models.User) error {
	if canManageUser(actor, target) {
		return nil
	}
	return errors.Forbiddenf("not authorized to manage this user")
}

func requireTenantAdministrator(actor *models.User) error {
	if isRootAdmin(actor) || isTenantAdmin(actor) {
		return nil
	}
	return errors.Forbiddenf("administrator role required")
}
