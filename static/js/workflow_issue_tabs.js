window.addEventListener('DOMContentLoaded', (event) => {
  document.getElementById('desk').addEventListener('tabselect', loadIssues);
  document.getElementById('needs-metadata').addEventListener('tabselect', loadIssues);
  document.getElementById('needs-review').addEventListener('tabselect', loadIssues);
  document.getElementById('unfixable-errors').addEventListener('tabselect', loadIssues);
});

function loadIssues(e) {
  const el = e.target;
  const panel = document.getElementById(el.getAttribute("aria-controls"));
  const tabid = el.getAttribute("id");
  console.log("Loading issues for " + el.getAttribute("id"));
  const deskFetcher = fetch(workflowHomeURL+"/json?tab="+tabid);
}
