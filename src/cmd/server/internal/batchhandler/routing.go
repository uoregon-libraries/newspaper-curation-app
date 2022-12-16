package batchhandler

import (
	"net/url"
	"path"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/web/tmpl"
)

var (
	basePath string
	conf     *config.Config

	// layout is the base template, cloned from the responder's layout, from
	// which all subpages are built
	layout *tmpl.TRoot

	// listTmpl is the template which shows all batches and actions
	listTmpl *tmpl.Template

	// viewTmpl is the batch view for showing details about a batch
	viewTmpl *tmpl.Template

	// approveFormTmpl is the (very simple) form to ensure QCer is certain they
	// want to push a batch to prod
	approveFormTmpl *tmpl.Template

	// rejectFormTmpl is the form for verifying a batch failed QC
	rejectFormTmpl *tmpl.Template

	// flagIssuesFormTmpl is the form for defining what needs to be fixed on a
	// batch which failed QC
	flagIssuesFormTmpl *tmpl.Template
)

func batchNewsURL(root string, b *Batch) string {
	var u, _ = url.Parse(root)
	u.Path = path.Join("batches", b.FullName())
	return u.String() + "/"
}

func batchURL(b *Batch, other ...string) string {
	var parts = []string{basePath, strconv.Itoa(b.ID)}
	if len(other) > 0 {
		parts = append(parts, other...)
	}
	return path.Join(parts...)
}

func flagIssuesURL(b *Batch) string {
	return batchURL(b, "flag-issues")
}

// Setup sets up all the routing rules and other configuration
func Setup(r *mux.Router, baseWebPath string, c *config.Config) {
	conf = c
	basePath = baseWebPath
	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(canView(listHandler))
	s.Path("/{batch_id}").Methods("GET").Handler(canView(viewHandler))
	s.Path("/{batch_id}/qc-ready").Methods("POST").Handler(canLoad(qcReadyHandler))
	s.Path("/{batch_id}/approve").Methods("GET").Handler(canApprove(qcApproveFormHandler))
	s.Path("/{batch_id}/approve").Methods("POST").Handler(canApprove(qcApproveHandler))
	s.Path("/{batch_id}/purge-staging").Methods("POST").Handler(canLoad(clearBatchStagingPurgeFlagHandler))
	s.Path("/{batch_id}/live").Methods("POST").Handler(canLoad(setLiveHandler))

	// All these paths are related to the same multi-step operation (rejecting a
	// batch, flagging issues, and finalizing it for rebuilding)
	s.Path("/{batch_id}/reject").Methods("GET").Handler(canReject(qcRejectFormHandler))
	s.Path("/{batch_id}/reject").Methods("POST").Handler(canReject(qcRejectHandler))
	s.Path("/{batch_id}/flag-issues").Methods("GET").Handler(canReject(flagIssuesFormHandler))
	s.Path("/{batch_id}/flag-issues").Methods("POST").Handler(canReject(flagIssuesHandler))

	layout = responder.Layout.Clone()
	layout.Funcs(tmpl.FuncMap{
		"BatchesHomeURL":    func() string { return basePath },
		"StagingRootURL":    func() string { return conf.StagingNewsWebroot },
		"ViewURL":           func(b *Batch) string { return batchURL(b) },
		"SetQCReadyURL":     func(b *Batch) string { return batchURL(b, "qc-ready") },
		"SetLiveURL":        func(b *Batch) string { return batchURL(b, "live") },
		"ApproveURL":        func(b *Batch) string { return batchURL(b, "approve") },
		"RejectURL":         func(b *Batch) string { return batchURL(b, "reject") },
		"ClearPurgeFlagURL": func(b *Batch) string { return batchURL(b, "purge-staging") },
		"FlagIssuesURL":     flagIssuesURL,
		"StagingBatchURL":   func(b *Batch) string { return batchNewsURL(conf.StagingNewsWebroot, b) },
		"ProdBatchURL":      func(b *Batch) string { return batchNewsURL(conf.NewsWebroot, b) },
	})
	layout.Path = path.Join(layout.Path, "batches")

	layout.MustReadPartials("_batch_metadata.go.html", "_load_purge.go.html")
	listTmpl = layout.MustBuild("list.go.html")
	viewTmpl = layout.MustBuild("view.go.html")
	approveFormTmpl = layout.MustBuild("approve_form.go.html")
	rejectFormTmpl = layout.MustBuild("reject_form.go.html")
	flagIssuesFormTmpl = layout.MustBuild("flag_issues_form.go.html")
}
