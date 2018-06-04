package userhandler

import (
	"cmd/server/internal/responder"
	"config"
	"db/user"
	"net/http"
	"path"
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
)

// Setup sets up all the routing rules and other configuration
func Setup(r *mux.Router, baseWebPath string, c *config.Config) {
	conf = c
	basePath = baseWebPath
	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(canView(listHandler))

	layout = responder.Layout.Clone()
	layout.Funcs(tmpl.FuncMap{"UsersHomeURL": func() string { return basePath }})
	layout.Path = path.Join(layout.Path, "users")
	listTmpl = layout.MustBuild("list.go.html")
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

// canView verifies the user can view the user list
func canView(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(user.ListUsers, h)
}
