package privilege

import (
	"testing"
)

func TestAllowedBy(t *testing.T) {
	var tests = map[string]struct {
		privilege *Privilege
		role      string
		want      bool
	}{
		"ListTitles allowed by any: sysop":          {ListTitles, "sysop", true},
		"ListTitles allowed by any: issue curator":  {ListTitles, "issue curator", true},
		"ModifyTitles allowed by title manager":     {ModifyTitles, "title manager", true},
		"ModifyTitles not allowed by issue curator": {ModifyTitles, "issue curator", false},
		"SysOp can do anything":                     {ModifyValidatedLCCNs, "sysop", true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var role = FindRole(tc.role)
			if role == nil {
				t.Fatalf("Role %q not found", tc.role)
			}

			var got = tc.privilege.AllowedBy(role)
			if got != tc.want {
				t.Errorf("AllowedBy(%q) = %#v, want %#v", tc.role, got, tc.want)
			}
		})
	}
}

func TestAllowedByAny(t *testing.T) {
	var tests = map[string]struct {
		privilege *Privilege
		roleNames []string
		want      bool
	}{
		"ListTitles allowed by any role":             {ListTitles, []string{"issue curator"}, true},
		"ListTitles allowed with no roles":           {ListTitles, []string{}, true},
		"ModifyUsers allowed by user manager":        {ModifyUsers, []string{"issue curator", "user manager"}, true},
		"ModifyUsers not allowed by unrelated roles": {ModifyUsers, []string{"issue curator", "batch loader"}, false},
		"ModifyUsers not allowed with no roles":      {ModifyUsers, []string{}, false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var roles = NewRoleSet()
			for _, name := range tc.roleNames {
				var role = FindRole(name)
				if role == nil {
					t.Fatalf("Role %q not found", name)
				}
				roles.Insert(role)
			}

			var got = tc.privilege.AllowedByAny(roles)
			if got != tc.want {
				t.Errorf("AllowedByAny(%#v) = %#v, want %#v", tc.roleNames, got, tc.want)
			}
		})
	}
}
