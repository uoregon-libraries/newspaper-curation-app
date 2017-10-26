package workflowhandler

import (
	"cmd/server/internal/responder"
	"net/http"
	"user"
)

// Alias the permission checks
func canView(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(user.ViewMetadataWorkflow, h)
}
func canWrite(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(user.EnterIssueMetadata, h)
}
func canReview(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(user.ReviewIssueMetadata, h)
}
