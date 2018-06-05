package userhandler

import (
	"cmd/server/internal/responder"
	"config"
	"db/user"
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

// canView verifies the user can view the user list
func canView(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(user.ListUsers, h)
}

// canModify verifies the user can create/edit/delete users
func canModify(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(user.ModifyUsers, h)
}
