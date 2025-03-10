package models

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
)

func getu() *User {
	var u = NewUser("test")
	u.RolesString = "title manager,user manager"
	u.deserialize()
	return u
}

func TestRoles(t *testing.T) {
	var u = getu()
	var roles = u.EffectiveRoles()
	if roles.Len() != 2 {
		t.Fatalf("Expected 2 roles, got %d", roles.Len())
	}
	if !roles.Contains(privilege.RoleTitleManager) {
		t.Errorf("Expected title manger to be in role list")
	}
	if !roles.Contains(privilege.RoleUserManager) {
		t.Errorf("Expected user manger to be in role list")
	}
}

func TestGrantExisting(t *testing.T) {
	var u = getu()
	var oldlen = u.EffectiveRoles().Len()
	u.Grant(privilege.RoleTitleManager)
	u.Grant(privilege.RoleTitleManager)
	var roles = u.EffectiveRoles()

	if roles.Len() != oldlen {
		t.Errorf("Granting an existing role shouldn't change the list (len was %d now %d)!", oldlen, roles.Len())
	}
}

func TestGrantNew(t *testing.T) {
	var u = getu()
	u.Grant(privilege.RoleMOCManager)
	u.serialize()
	var want = "marc org code manager,title manager,user manager"
	if u.RolesString != want {
		t.Errorf("Granting a new role should update the roles string (got %q, want %q)", u.RolesString, want)
	}

	var roles = u.EffectiveRoles()
	if roles.Len() != 3 {
		t.Errorf("Granting a new role should update the roles list (length should be 3; got %d)", roles.Len())
	}
}

func TestEffectiveRoles(t *testing.T) {
	var everything = privilege.AssignableRoles()
	var nosysop = everything.Clone()
	nosysop.Remove(privilege.RoleSysOp)

	var tests = map[string]struct {
		roles string
		want  []string
	}{
		"sysop has everything":           {"sysop", everything.Names()},
		"site manager has all but sysop": {"site manager", nosysop.Names()},
		"basic user":                     {"issue curator,batch loader", []string{"batch loader", "issue curator"}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var u = NewUser("test")
			u.RolesString = tc.roles
			u.deserialize()
			var got = u.EffectiveRoles().Names()
			var diff = cmp.Diff(tc.want, got)

			if diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

func TestDeny(t *testing.T) {
	var u = getu()
	var rs = u.RolesString
	u.Deny(privilege.RoleMOCManager)
	u.serialize()
	if u.RolesString != rs {
		t.Errorf("Denying a not-granted role shouldn't change anything (got %q)", u.RolesString)
	}

	u.Deny(privilege.RoleTitleManager)
	u.serialize()
	if u.RolesString == rs {
		t.Errorf("Denying title manager should remove it (got %q)", u.RolesString)
	}
	if u.EffectiveRoles().Len() != 1 {
		t.Errorf("Denying title manager should remove it (got %d roles)", u.EffectiveRoles().Len())
	}

	u.Grant(privilege.RoleMOCManager)
	u.Grant(privilege.RoleIssueCurator)
	u.Deny(privilege.RoleMOCManager)
	u.Deny(privilege.RoleIssueCurator)
	u.Deny(privilege.RoleUserManager)
	u.serialize()

	if u.RolesString != "" {
		t.Errorf("Deny should remove roles (got %q)", u.RolesString)
	}
	if u.EffectiveRoles().Len() != 0 {
		t.Errorf("Deny should remove roles (got %d roles)", u.EffectiveRoles().Len())
	}
}

func TestCanGrant(t *testing.T) {
	// We don't use `getu` here because we want to know precisely which roles
	// we're testing
	var u = NewUser("test")
	u.RolesString = "title manager,user manager"
	u.deserialize()

	for _, r := range []*privilege.Role{privilege.RoleTitleManager, privilege.RoleUserManager} {
		if !u.CanGrant(r) {
			t.Error("User manager should be allowed to grant any assigned roles")
		}
	}

	if u.CanGrant(privilege.RoleIssueCurator) {
		t.Error("User manager shouldn't be allowed to grant unassigned roles")
	}

	u.realRoles.Empty()
	if u.PermittedTo(privilege.ModifyUsers) {
		t.Error("User with no roles shouldn't be allowed to modify users")
	}

	u.Grant(privilege.RoleIssueCurator)
	if u.CanGrant(privilege.RoleIssueCurator) {
		t.Error("Non-user-manager shouldn't be allowed to grant any roles")
	}

	// We have to set a role via string, then deserialize here because
	// deserializing is where we apply implicit roles
	u.RolesString = "sysop"
	u.deserialize()
	if !u.CanGrant(privilege.RoleUserManager) {
		t.Errorf("SysOp should be allowed to grant user manager role")
	}
}
