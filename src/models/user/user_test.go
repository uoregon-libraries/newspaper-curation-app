package user

import (
	"testing"
)

func getu() *User {
	var u = &User{Login: "test", RolesString: "title manager,user manager"}
	u.deserialize()
	return u
}

func TestRoles(t *testing.T) {
	var u = getu()
	var roles = u.Roles()
	if len(roles) != 2 {
		t.Fatalf("Expected 2 roles, got %d", len(roles))
	}
	if roles[0] != RoleTitleManager {
		t.Errorf("Expected first role to be title manger; got %q", roles[0].Name)
	}
	if roles[1] != RoleUserManager {
		t.Errorf("Expected second role to be user manger; got %q", roles[1].Name)
	}
}

func TestGrantExisting(t *testing.T) {
	var u = getu()
	var oldlen = len(u.Roles())
	u.Grant(RoleTitleManager)
	u.Grant(RoleTitleManager)
	var roles = u.Roles()

	if len(roles) != oldlen {
		t.Errorf("Granting an existing role shouldn't change the list (len was %d now %d)!", oldlen, len(roles))
	}
}

func TestGrantReserializes(t *testing.T) {
	var u = getu()
	u.Grant(RoleMOCManager)
	if u.RolesString != "title manager,user manager,marc org code manager" {
		t.Errorf("Granting a new role should update the roles string (got %q)", u.RolesString)
	}

	var roles = u.Roles()
	if len(roles) != 3 {
		t.Errorf("Granting a new role should update the roles list (length should be 3; got %d)", len(roles))
	}
}

func TestDeny(t *testing.T) {
	var u = getu()
	var rs = u.RolesString
	u.Deny(RoleMOCManager)
	if u.RolesString != rs {
		t.Errorf("Denying a not-granted role shouldn't change anything (got %q)", u.RolesString)
	}

	u.Deny(RoleTitleManager)
	if u.RolesString == rs {
		t.Errorf("Denying title manager should remove it (got %q)", u.RolesString)
	}
	if len(u.Roles()) != 1 {
		t.Errorf("Denying title manager should remove it (got %d roles)", len(u.Roles()))
	}

	u.Grant(RoleMOCManager)
	u.Grant(RoleIssueCurator)
	u.Deny(RoleMOCManager)
	u.Deny(RoleIssueCurator)
	u.Deny(RoleUserManager)

	if u.RolesString != "" {
		t.Errorf("Deny should remove roles and reserialize (got %q)", u.RolesString)
	}
	if len(u.roles) != 0 {
		t.Errorf("Deny should remove roles and reserialize (got %d roles)", len(u.roles))
	}
}

func TestCanGrant(t *testing.T) {
	var u = getu()
	for _, r := range u.roles {
		if !u.CanGrant(r) {
			t.Error("User manager should be allowed to grant any assigned roles")
		}
	}

	if u.CanGrant(RoleIssueCurator) {
		t.Error("User manager shouldn't be allowed to grant unassigned roles")
	}

	for _, r := range u.roles {
		u.Deny(r)
	}

	if u.PermittedTo(ModifyUsers) {
		t.Error("User with no roles shouldn't be allowed to modify users")
	}

	u.Grant(RoleIssueCurator)
	if u.CanGrant(u.roles[0]) {
		t.Error("Non-user-manager shouldn't be allowed to grant any roles")
	}

	u.roles = nil
	u.Grant(RoleAdmin)
	if !u.CanGrant(RoleUserManager) {
		t.Errorf("Admin should be allowed to grant user manager role")
	}
}
