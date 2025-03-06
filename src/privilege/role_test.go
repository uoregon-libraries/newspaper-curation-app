package privilege

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFindRole(t *testing.T) {
	var tests = map[string]struct {
		roleName string
		wantNil  bool
	}{
		"existing role":         {"title manager", false},
		"another existing role": {"issue curator", false},
		"non-existent role":     {"nonexistent role", true},
		"empty role name":       {"", true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var role = FindRole(tc.roleName)
			if (role == nil) != tc.wantNil {
				t.Errorf("FindRole(%q) = %v, want nil: %v", tc.roleName, role, tc.wantNil)
			}
			if !tc.wantNil && role.Name != tc.roleName {
				t.Errorf("FindRole(%q).Name = %q, want %q", tc.roleName, role.Name, tc.roleName)
			}
		})
	}
}

func TestTitle(t *testing.T) {
	var tests = []struct {
		roleName string
		want     string
	}{
		{"title manager", "Title Manager"},
		{"issue curator", "Issue Curator"},
		{"marc org code manager", "MARC Org Code Manager"},
		{"sysop", "Sysop"},
	}

	for _, tc := range tests {
		t.Run(tc.roleName+" title", func(t *testing.T) {
			var role = FindRole(tc.roleName)
			if role == nil {
				t.Fatalf("Role %q not found", tc.roleName)
			}
			var got = role.Title()
			if got != tc.want {
				t.Errorf("Role(%q).Title() = %q, want %q", tc.roleName, got, tc.want)
			}
		})
	}
}

func TestPrivileges(t *testing.T) {
	// Test that each role has some privileges
	var roles = []string{
		"sysop",
		"site manager",
		"title manager",
		"issue curator",
		"issue reviewer",
		"issue manager",
		"user manager",
		"marc org code manager",
		"workflow manager",
		"batch builder",
		"batch reviewer",
		"batch loader",
	}

	for _, roleName := range roles {
		t.Run(roleName+" privileges", func(t *testing.T) {
			var role = FindRole(roleName)
			if role == nil {
				t.Fatalf("Role %q not found", roleName)
			}

			var privs = role.Privileges()

			// SysOp should have all privileges
			if roleName == "sysop" && len(privs) != len(Privileges) {
				t.Errorf("SysOp should have all privileges, got %d, want %d",
					len(privs), len(Privileges))
			}

			// Other roles should have at least one privilege (except "any")
			if roleName != "-any-" && len(privs) == 0 {
				t.Errorf("Role %q has no privileges", roleName)
			}
		})
	}
}

func TestRoleSet(t *testing.T) {
	t.Run("NewRoleSet", func(t *testing.T) {
		var sysop = FindRole("sysop")
		var curator = FindRole("issue curator")

		var rs = NewRoleSet(sysop, curator)

		if !rs.Contains(sysop) {
			t.Error("RoleSet should contain sysop")
		}
		if !rs.Contains(curator) {
			t.Error("RoleSet should contain curator")
		}
		if rs.Contains(FindRole("user manager")) {
			t.Error("RoleSet should not contain user manager")
		}
	})

	t.Run("Insert and Remove", func(t *testing.T) {
		var rs = NewRoleSet()
		var manager = FindRole("user manager")

		rs.Insert(manager)
		if !rs.Contains(manager) {
			t.Error("After Insert, RoleSet should contain role")
		}

		rs.Remove(manager)
		if rs.Contains(manager) {
			t.Error("After Remove, RoleSet should not contain role")
		}
	})

	t.Run("Union", func(t *testing.T) {
		var rs1 = NewRoleSet(
			FindRole("sysop"),
			FindRole("site manager"),
		)
		var rs2 = NewRoleSet(
			FindRole("user manager"),
			FindRole("site manager"), // Duplicate
		)

		var union = rs1.Union(rs2)

		if !union.Contains(FindRole("sysop")) {
			t.Error("Union should contain sysop from rs1")
		}
		if !union.Contains(FindRole("site manager")) {
			t.Error("Union should contain site manager")
		}
		if !union.Contains(FindRole("user manager")) {
			t.Error("Union should contain user manager from rs2")
		}
		var got = union.Names()
		var expected = []string{"site manager", "sysop", "user manager"}

		var diff = cmp.Diff(got, expected)
		if diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("Empty", func(t *testing.T) {
		var rs = NewRoleSet(
			FindRole("sysop"),
			FindRole("site manager"),
		)

		rs.Empty()

		if len(rs.items) != 0 {
			t.Errorf("After Empty(), RoleSet should have no roles, got %v", rs.Names())
		}
	})
}
