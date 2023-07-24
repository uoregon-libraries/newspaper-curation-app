package userhandler

import (
	"fmt"
	"html/template"
	"net/http"
	"path"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
	"github.com/uoregon-libraries/newspaper-curation-app/src/web/tmpl"
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
	s.Path("/deactivate").Methods("POST").Handler(canModify(deactivateHandler))

	layout = responder.Layout.Clone()
	layout.Funcs(tmpl.FuncMap{
		"UsersHomeURL": func() string { return basePath },
		"Roles":        func() []*privilege.Role { return privilege.AssignableRoles },
	})
	layout.Path = path.Join(layout.Path, "users")

	listTmpl = layout.MustBuild("list.go.html")
	formTmpl = layout.MustBuild("form.go.html")
}

func getUserForModify(r *responder.Responder) (u *models.User, handled bool) {
	var idStr = r.Request.FormValue("id")
	var id, _ = strconv.ParseInt(idStr, 10, 64)
	if id < 1 {
		logger.Warnf("Invalid user id for request %q (%s)", r.Request.URL.Path, idStr)
		r.Error(http.StatusBadRequest, "Invalid user id - try again or contact support")
		return nil, true
	}

	u = models.FindUserByID(id)
	if u == models.EmptyUser {
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
	var users, err = models.ActiveUsers()
	if err != nil {
		logger.Errorf("Unable to load user list: %s", err)
		r.Error(http.StatusInternalServerError, "Error trying to pull user list - try again or contact support")
		return
	}

	// Non-admins don't see admins at all
	if !r.Vars.User.IsAdmin() {
		var nonAdmins []*models.User
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
	r.Vars.Data["User"] = models.NewUser("")
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

// getUserForSave uses the "id" param to determine if this is a create or an
// update, sets up a User, applies roles based on the form, and returns the
// user instance.  If anything goes wrong, an error will be sent to the client
// and "handled" will be true, alerting the caller not to process anything.
func getUserForSave(r *responder.Responder) (u *models.User, handled bool) {
	// If this is an update, we need to load the original user first
	if r.Request.FormValue("id") != "" {
		u, handled = getUserForModify(r)
		if handled {
			return nil, true
		}
	} else {
		var login = r.Request.FormValue("login")
		u = models.NewUser(login)
	}

	return u, applyRoles(r, u)
}

// applyRoles parses the form to figure out grants / denies and sets them on
// the given user.  Returns handled == true if there are any critical errors
// which should prevent further processing.
func applyRoles(r *responder.Responder, u *models.User) (handled bool) {
	var err = r.Request.ParseForm()
	if err != nil {
		logger.Errorf("Unable to parse the form when trying to save a user: %s", err)
		r.Error(http.StatusInternalServerError, "Error trying to save user data - try again or contact support")
		return true
	}

	for param, vlist := range r.Request.Form {
		if len(param) > 5 && param[:5] == "role-" {
			var roleName = param[5:]
			var role = privilege.FindRole(roleName)

			// This shouldn't be possible *unless* I screwed up the form, in which case the user can't do much
			if role == nil {
				logger.Errorf("Invalid role %q in user editor", roleName)
				r.Error(http.StatusInternalServerError, "Error trying to save user data - try again or contact support")
				return true
			}

			// This shouldn't be possible unless the user is trying to hack the form
			if !r.Vars.User.CanGrant(role) {
				logger.Errorf("User %q trying to set (or remove) an unpermitted role (%q) in user editor",
					r.Vars.User.Login, role.Name)
				r.Error(http.StatusUnauthorized, "You are not permitted to set this role")
				return true
			}

			// For safety we only accept one or zero values; anything else is silently thrown away
			if vlist[0] == "1" {
				u.Grant(role)
			}

			if vlist[0] == "0" {
				u.Deny(role)
			}
		}
	}

	return false
}

// handleInvalidUser verifies that a new user's login isn't blank and isn't a
// dupe.  Currently there's no way to have an error if the user isn't new - we
// already validated the roles are grantable, and you can't edit a user's
// login.  But we want *all* saves to use this function just in case validation
// changes in the future.
func handleInvalidUser(r *responder.Responder, u *models.User) (handled bool) {
	if u.ID != 0 {
		return false
	}

	if u.Login == "" {
		r.Vars.Alert = template.HTML("Cannot create a user with no login name")
		handled = true
	} else if models.FindActiveUserWithLogin(u.Login) != models.EmptyUser {
		r.Vars.Alert = template.HTML("User " + u.Login + " already exists")
		handled = true
	}

	if handled {
		r.Vars.Data["User"] = u
		r.Vars.Title = "Create a new user"
		r.Render(formTmpl)
		return true
	}

	return false
}

// saveHandler inserts or updates a user in the db, translating the checkbox
// values to Grant/Deny calls
func saveHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var u, handled = getUserForSave(r)
	if handled {
		return
	}

	// We validate the login after getting all the roles set - this allows us to
	// redisplay the form with all the role data filled out
	if handleInvalidUser(r, u) {
		return
	}

	var err = u.Save()
	if err != nil {
		logger.Errorf("Unable to save user %q: %s", u.Login, err)
		r.Error(http.StatusInternalServerError, "Error trying to save user data - try again or contact support")
		return
	}

	r.Audit(models.AuditActionSaveUser, fmt.Sprintf("Login: %q, roles: %q", u.Login, u.RolesString))
	http.SetCookie(w, &http.Cookie{Name: "Info", Value: "User data saved", Path: "/"})
	http.Redirect(w, req, basePath, http.StatusFound)
}

// deactivateHandler removes the given user from the db
func deactivateHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var u, handled = getUserForModify(r)
	if handled {
		return
	}

	// Make sure the current user can actually edit the loaded user
	if !r.Vars.User.CanModifyUser(u) {
		r.Error(http.StatusUnauthorized, "You are not allowed to deactivate this user")
		return
	}

	var err = u.Deactivate()
	if err != nil {
		logger.Errorf("Unable to deactivate user (id %d): %s", u.ID, err)
		r.Error(http.StatusInternalServerError, "Error trying to deactivate user - try again or contact support")
		return
	}

	r.Audit(models.AuditActionDeactivateUser, u.Login)
	http.SetCookie(w, &http.Cookie{Name: "Info", Value: fmt.Sprintf("Deactivated user '%s'", u.Login), Path: "/"})
	http.Redirect(w, req, basePath, http.StatusFound)
}

// canView verifies the user can view the user list
func canView(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(privilege.ListUsers, h)
}

// canModify verifies the user can create/edit/deactivate users
func canModify(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(privilege.ModifyUsers, h)
}
