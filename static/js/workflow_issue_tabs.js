window.addEventListener('DOMContentLoaded', (event) => {
  // Preserve the original caption for all tables
  document.querySelectorAll('table').forEach((table) => {
    table.dataset.caption = table.caption.innerText;
  });

  document.getElementById('desk').addEventListener('tabselect', loadIssues);
  document.getElementById('needs-metadata').addEventListener('tabselect', loadIssues);
  document.getElementById('needs-review').addEventListener('tabselect', loadIssues);
  document.getElementById('unfixable-errors').addEventListener('tabselect', loadIssues);
});

async function loadIssues(e) {
  // Start by clearing the table caption and body in this tab
  const el = e.target;
  const panel = document.getElementById(el.getAttribute('aria-controls'));
  const table = panel.querySelector('table');
  const emptyDiv = panel.querySelector('.empty');
  emptyDiv.setAttribute('hidden', true);
  table.caption.innerText = 'Loading...';
  for (i = 0; i < table.tBodies.length; i++) {
    table.tBodies.item(i).outerHTML = '';
  };

  const statusDiv = document.getElementById('json-status');
  statusDiv.setAttribute('class', 'alert alert-info');
  statusDiv.innerText = 'Fetching issues from server...';

  var response, data;
  try {
    response = await fetch(workflowHomeURL+'/json?tab='+el.getAttribute('id'));
    data = await response.json();
  }
  catch (e) {
    console.log('Exception caught: ' + e);
    statusDiv.setAttribute('class', 'alert alert-danger');
    statusDiv.innerText = 'Network error trying to retrieve issues: please reload the page and try again, or contact support.';
    return;
  }

  if (!response.ok) {
    statusDiv.setAttribute('class', 'alert alert-warning');
    statusDiv.innerText = data.Message;
    return;
  }

  statusDiv.setAttribute('class', 'alert alert-success');
  statusDiv.innerText = 'Load complete';
  if (data.Issues == null || data.Issues.length == 0) {
    table.setAttribute('hidden', true);
    emptyDiv.removeAttribute('hidden');
    return
  }

  populateTable(table, data.Issues);
  table.removeAttribute('hidden');
  emptyDiv.setAttribute('hidden', true);
}

function populateTable(table, issues) {
  table.caption.innerText = table.dataset.caption;

  const tBody = table.createTBody();
  var row, cell;
  for (i = 0; i < issues.length; i++) {
    let issue = issues[i];
    row = tBody.insertRow();
    // In all cases, cell 1 is the publication's title and cell 2 is the issue date
    cell = document.createElement('th');
    cell.setAttribute('scope', 'row');
    cell.innerText = `${issue.Title} (${issue.LCCN})`;
    row.appendChild(cell);
    cell = document.createElement('th');
    cell.setAttribute('scope', 'row');
    cell.innerText = issue.Date;
    row.appendChild(cell);

    // If we're on the desk, we have a third cell before actions for the workflow status and expiration
    if (table.parentNode.getAttribute('id') == 'desk-tab') {
      cell = row.insertCell();
      cell.innerHTML = issue.Task + '<br />' + 'Expires on ' + issue.Expiration;
    }

    // Finally, build out all the actions....
    cell = row.insertCell();
    for (j = 0; j < issue.Actions.length; j++) {
      cell.innerHTML += buildActionHTML(issue.Actions[j]);
    }
  }
}

function buildActionHTML(action) {
  if (action.Type == 'link') {
    return `<a href="${action.Path}" class="btn btn-default">${action.Text}</a>`;
  }

  let classname = (action.Type == 'button-danger') ? 'btn-danger' : 'btn-primary';
  return `
  <form action="${action.Path}" method="POST" class="actions">
    <button type="submit" class="btn ${classname}">${action.Text}</button>
  </form>
  `;
}
