package userhandler

import (
	"cmd/server/internal/responder"
	"config"
	"db/user"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"strconv"
	"web/tmpl"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/gopkg/logger"
)

var (
	basePath string
	conf     *config.Config

	// layout is the base template, cloned from the responder's layout, from
	// which all subpages are built
	layout *tmpl.TRoot

	// listTmpl is the template which shows all users
	listTmpl *tmpl.Template

	// formTmpl is the form for adding or editing a user
	formTmpl *tmpl.Template
)

// Setup sets up all the routing rules and other configuration
func Setup(r *mux.Router, baseWebPath string, c *config.Config) {
	conf = c
	basePath = baseWebPath
	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(canView(listHandler))
	s.Path("/new").Handler(canModify(newHandler))
	s.Path("/edit").Handler(canModify(editHandler))
	s.Path("/save").Methods("POST").Handler(canModify(saveHandler))
	s.Path("/delete").Methods("POST").Handler(canModify(deleteHandler))

	layout = responder.Layout.Clone()
	layout.Funcs(tmpl.FuncMap{
		"UsersHomeURL": func() string { return basePath },
		"Roles":        func() []*user.Role { return user.AssignableRoles },
	})
	layout.Path = path.Join(layout.Path, "users")

	listTmpl = layout.MustBuild("list.go.html")
	formTmpl = layout.MustBuild("form.go.html")
}

func getUserForModify(r *responder.Responder) (u *user.User, handled bool) {
	var idStr = r.Request.FormValue("id")
	var id, _ = strconv.Atoi(idStr)
	if id < 1 {
		logger.Warnf("Invalid user id for request %q (%s)", r.Request.URL.Path, idStr)
		r.Error(http.StatusBadRequest, "Invalid user id - try again or contact support")
		return nil, true
	}

	u = user.FindByID(id)
	if u == user.EmptyUser {
		r.Error(http.StatusNotFound, "Unable to find user - try again or contact support")
		return nil, true
	}

	if !r.Vars.User.CanModifyUser(u) {
		logger.Errorf("User %q trying to modify a user they shouldn't be able to (user %q, id %d)",
			r.Vars.User.Login, u.Login, u.ID)
		r.Error(http.StatusUnauthorized, "You are not permitted to edit this user")
		return nil, true
	}

	return u, false
}

// listHandler spits out the list of users
func listHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	r.Vars.Title = "Users"
	var users, err = user.All()
	if err != nil {
		logger.Errorf("Unable to load user list: %s", err)
		r.Error(http.StatusInternalServerError, "Error trying to pull user list - try again or contact support")
		return
	}

	// Non-admins don't see admins at all
	if !r.Vars.User.IsAdmin() {
		var nonAdmins []*user.User
		for _, u := range users {
			if !u.IsAdmin() {
				nonAdmins = append(nonAdmins, u)
			}
		}
		users = nonAdmins
	}

	r.Vars.Data["Users"] = users
	r.Render(listTmpl)
}

// newHandler shows a form for adding a new user
func newHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	r.Vars.Data["User"] = user.New("")
	r.Vars.Title = "Create a new user"
	r.Render(formTmpl)
}

// editHandler loads the user by id and renders the edit form.  Users are not
// allowed to edit themselves or admins.
func editHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var u, handled = getUserForModify(r)
	if handled {
		return
	}

	r.Vars.Data["User"] = u
	r.Vars.Title = "Editing " + u.Login
	r.Render(formTmpl)
}

// saveHandler inserts or updates a user in the db, translating the checkbox
// values to Grant/Deny calls
func saveHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var u *user.User

	// If this is an update, we need to load the original user first
	if r.Request.FormValue("id") != "" {
		var handled bool
		u, handled = getUserForModify(r)
		if handled {
			return
		}
	} else {
		var login = r.Request.FormValue("login")
		u = user.New(login)
	}

	// Parse form to figure out grants / denies
	var err = r.Request.ParseForm()
	if err != nil {
		logger.Errorf("Unable to parse the form when trying to save a user: %s", err)
		r.Error(http.StatusInternalServerError, "Error trying to save user data - try again or contact support")
		return
	}

	for param, vlist := range r.Request.Form {
		if len(param) > 5 && param[:5] == "role-" {
			var roleName = param[5:]
			var role = user.FindRole(roleName)

			// This shouldn't be possible *unless* I screwed up the form, in which case the user can't do much
			if role == nil {
				logger.Errorf("Invalid role %q in user editor", roleName)
				r.Error(http.StatusInternalServerError, "Error trying to save user data - try again or contact support")
				return
			}

			// This shouldn't be possible unless the user is trying to hack the form
			if !r.Vars.User.CanGrant(role) {
				logger.Errorf("User %q trying to set (or remove) an unpermitted role (%q) in user editor",
					r.Vars.User.Login, role.Name)
				r.Error(http.StatusUnauthorized, "You are not permitted to set this role")
				return
			}

			if vlist[0] == "1" {
				u.Grant(role)
			}

			if vlist[0] == "0" {
				u.Deny(role)
			}
		}
	}

	// We validate the login after getting all the roles set - this allows us to
	// redisplay the form with all the role data filled out
	if u.ID == 0 {
		var errored bool
		if u.Login == "" {
			r.Vars.Alert = template.HTML("Cannot create a user with no login name")
			errored = true
		} else if user.FindByLogin(u.Login) != user.EmptyUser {
			r.Vars.Alert = template.HTML("User " + u.Login + " already exists")
			errored = true
		}

		if errored {
			r.Vars.Data["User"] = u
			r.Vars.Title = "Create a new user"
			r.Render(formTmpl)
			return
		}
	}

	err = u.Save()
	if err != nil {
		logger.Errorf("Unable to save user %q: %s", u.Login, err)
		r.Error(http.StatusInternalServerError, "Error trying to save user data - try again or contact support")
		return
	}

	r.Audit("save-user", fmt.Sprintf("Login: %q, roles: %q", u.Login, u.RolesString))
	http.SetCookie(w, &http.Cookie{Name: "Info", Value: "User data saved", Path: "/"})
	http.Redirect(w, req, basePath, http.StatusFound)
}

// deleteHandler removes the given user from the db
func deleteHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var u, handled = getUserForModify(r)
	if handled {
		return
	}

	// Make sure the current user can actually edit the loaded user
	if !r.Vars.User.CanModifyUser(u) {
		r.Error(http.StatusUnauthorized, "You are not allowed to delete this user")
		return
	}

	var err = u.Delete()
	if err != nil {
		logger.Errorf("Unable to delete user (id %d): %s", u.ID, err)
		r.Error(http.StatusInternalServerError, "Error trying to delete user - try again or contact support")
		return
	}

	r.Audit("delete-user", u.Name)
	http.SetCookie(w, &http.Cookie{Name: "Info", Value: "Deleted user", Path: "/"})
	http.Redirect(w, req, basePath, http.StatusFound)
}

// canView verifies the user can view the user list
func canView(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(user.ListUsers, h)
}

// canModify verifies the user can create/edit/delete users
func canModify(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(user.ModifyUsers, h)
}
