package batchhandler

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/jobs"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// listHandler spits out the list of batches
func listHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	r.Vars.Title = "Batches"
	var list, err = models.ActionableBatches()
	if err != nil {
		logger.Criticalf("Unable to load batches: %s", err)
		r.Error(http.StatusInternalServerError, "Error trying to pull batch list - try again or contact support")
		return
	}

	var wrapped []*Batch
	wrapped, err = wrapBatches(list, r.Vars.User)
	if err != nil {
		logger.Criticalf("Unable to wrap batches: %s", err)
		r.Error(http.StatusInternalServerError, "Error trying to pull batch list - try again or contact support")
		return
	}

	// Break batches into groups: not live, live, archived but not complete, and
	// completed. The latter is its own group because it's a huge list that, most
	// of the time, we don't need to browse.
	var inproc, live, archived, complete []*Batch
	for _, b := range wrapped {
		// Immediately skip anything the current user can't even view
		if !b.Can().View() {
			continue
		}

		if !b.StatusMeta.Live {
			inproc = append(inproc, b)
		} else {
			switch b.Status {
			case models.BatchStatusLive:
				live = append(live, b)
			case models.BatchStatusLiveArchived:
				archived = append(archived, b)
			case models.BatchStatusLiveDone:
				complete = append(complete, b)
			}
		}
	}

	r.Vars.Data["InProcess"] = inproc
	r.Vars.Data["Live"] = live
	r.Vars.Data["Archived"] = archived
	r.Vars.Data["Complete"] = complete
	r.Render(listTmpl)
}

func viewHandler(w http.ResponseWriter, req *http.Request) {
	var r, ok = getBatchResponder(w, req)
	if !ok {
		return
	}
	if !r.batch.Can().View() {
		r.Error(http.StatusForbidden, "You are not permitted to view this batch")
		return
	}
	r.Vars.Title = fmt.Sprintf("Viewing batch (%s)", r.batch.Name)
	r.Render(viewTmpl)
}

func qcApproveFormHandler(w http.ResponseWriter, req *http.Request) {
	var r, ok = getBatchResponder(w, req)
	if !ok {
		return
	}
	if !r.batch.Can().Approve() {
		r.Error(http.StatusForbidden, "You are not permitted to approve this batch for a production load")
		return
	}

	r.Vars.Title = "Approve batch?"
	r.Render(approveFormTmpl)
}

func qcApproveHandler(w http.ResponseWriter, req *http.Request) {
	var r, ok = getBatchResponder(w, req)
	if !ok {
		return
	}
	if !r.batch.Can().Approve() {
		r.Error(http.StatusForbidden, "You are not permitted to approve this batch for a production load")
		return
	}

	// TODO: send job to ONI to load batch live
	var err = r.batch.Save(models.ActionTypeApproveBatch, r.Vars.User.ID, "")
	if err != nil {
		logger.Criticalf(`Unable to log "approve batch" action for batch %d (%s): %s`, r.batch.ID, r.batch.FullName, err)
	} else {
		err = jobs.QueueBatchGoLive(r.batch.Batch, conf)
		if err != nil {
			logger.Criticalf(`Unable to queue go-live job for batch %d (%s): %s`, r.batch.ID, r.batch.FullName, err)
		}
	}

	// If either operation above gave an error, fully reset the batch so we can
	// re-render without risk the job queue did something weird
	if err != nil {
		r, ok = getBatchResponder(w, req)
		if !ok {
			return
		}

		r.Vars.Title = `Error approving batch`
		r.Vars.Alert = template.HTML(`Unable to approve this batch. Try again or contact support.`)
		r.Render(viewTmpl)
		return
	}

	http.SetCookie(r.Writer, &http.Cookie{Name: "Info", Value: r.batch.Name + ": approved for production load", Path: "/"})
	http.Redirect(w, req, basePath, http.StatusFound)
}

func setArchivedHandler(w http.ResponseWriter, req *http.Request) {
	var r, ok = getBatchResponder(w, req)
	if !ok {
		return
	}
	if !r.batch.Can().Archive() {
		r.Error(http.StatusForbidden, "You are not permitted to flag batches as having been archived")
		return
	}

	r.batch.Status = models.BatchStatusLiveArchived
	r.batch.ArchivedAt = time.Now()
	var err = r.batch.SaveWithoutAction()
	if err != nil {
		logger.Criticalf(`Unable to flag batch %d (%s) as archived: %s`, r.batch.ID, r.batch.FullName, err)

		// Reload the batch and rerender
		r, ok = getBatchResponder(w, req)
		if !ok {
			return
		}

		r.Vars.Title = `Error marking batch "archived"`
		r.Vars.Alert = template.HTML(`Unable to set batch as "archived". Try again or contact support.`)
		r.Render(viewTmpl)
		return
	}

	http.SetCookie(r.Writer, &http.Cookie{Name: "Info", Value: r.batch.Name + ": marked batch as 'archived'", Path: "/"})
	http.Redirect(w, req, basePath, http.StatusFound)
}

func qcRejectFormHandler(w http.ResponseWriter, req *http.Request) {
	var r, ok = getBatchResponder(w, req)
	if !ok {
		return
	}
	if !r.batch.Can().Reject() {
		r.Error(http.StatusForbidden, "You are not permitted to reject this batch")
		return
	}

	r.Vars.Title = "Reject batch?"
	r.Render(rejectFormTmpl)
}

func qcRejectHandler(w http.ResponseWriter, req *http.Request) {
	var r, ok = getBatchResponder(w, req)
	if !ok {
		return
	}
	if !r.batch.Can().Reject() {
		r.Error(http.StatusForbidden, "You are not permitted to reject this batch")
		return
	}

	var oldStatus = r.batch.Status
	r.batch.Status = models.BatchStatusQCFlagIssues
	var err = r.batch.Save(models.ActionTypeRejectBatch, r.Vars.User.ID, "")
	if err != nil {
		// Since we're merely re-rending the template, we must put the batch back
		// to its original state or the template could be weird/broken
		r.batch.Status = oldStatus
		logger.Criticalf("Unable to reject batch %d (%s): %s", r.batch.ID, r.batch.FullName, err)
		r.Vars.Title = "Error saving batch"
		r.Vars.Alert = template.HTML("Unable to reject batch. Try again or contact support.")
		r.Render(rejectFormTmpl)
		return
	}

	http.SetCookie(r.Writer, &http.Cookie{Name: "Info", Value: r.batch.Name + ": failed QC, ready to flag issues for removal", Path: "/"})
	http.Redirect(w, req, batchURL(r.batch), http.StatusFound)
}
