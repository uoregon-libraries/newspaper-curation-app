window.addEventListener('DOMContentLoaded', (event) => {
  // Preserve the original caption for all tables
  document.querySelectorAll('table').forEach((table) => {
    table.dataset.caption = table.caption.innerText;
  });

  // Read the URL to determine if we already have a filter in place - this must
  // happen prior to any tabs being loaded
  setFilterValuesFromURL();

  // Add on-select listeners to pull issues from the server whenever a new tab
  // is selected
  document.getElementById('desk').addEventListener('tabselect', loadIssues);
  document.getElementById('needs-metadata').addEventListener('tabselect', loadIssues);
  document.getElementById('needs-review').addEventListener('tabselect', loadIssues);
  document.getElementById('unfixable-errors').addEventListener('tabselect', loadIssues);

  // Set up the filter form to fetch JSON from the server on submit
  document.getElementById('filter-form').addEventListener('submit', applyFilter);
});

function setFilterValuesFromURL() {
  let u = new URL(window.location);
  let srch = new URLSearchParams(u.search.substr(1));
  document.getElementById('lccn').value = srch.get('lccn');
  document.getElementById('moc').value = srch.get('moc');
}

function applyFilter(e) {
  let u = new URL(window.location);
  let srch = new URLSearchParams(u.search.substr(1));
  srch.set('lccn', document.getElementById('lccn').value);
  srch.set('moc', document.getElementById('moc').value);
  u.search = srch.toString();
  history.replaceState(null, "", u);
  loadTabIssues(document.getElementById(srch.get('tab')));

  e.preventDefault();
}

function loadIssues(e) {
  loadTabIssues(e.target);
}

async function loadTabIssues(tab) {
  // Start by clearing the table caption and body in this tab
  const panel = document.getElementById(tab.getAttribute('aria-controls'));
  const table = panel.querySelector('table');
  const emptyDiv = panel.querySelector('.empty');
  emptyDiv.setAttribute('hidden', true);
  table.caption.innerText = 'Loading...';
  for (i = 0; i < table.tBodies.length; i++) {
    table.tBodies.item(i).outerHTML = '';
  };

  const statusDiv = document.getElementById('json-status');
  let loading = setTimeout(() => {
    statusDiv.setAttribute('class', 'alert alert-info');
    statusDiv.dataset.faded = false;
    statusDiv.innerText = 'Fetching issues from server...';
  }, 200);

  let u = new URL(window.location);
  let srch = new URLSearchParams(u.search.substr(1));

  var response, data;
  try {
    response = await fetch(workflowHomeURL+'/json?'+srch.toString());
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

  // Refresh all tabs' counts
  document.querySelectorAll('[role="tab"]').forEach((el) => {
    el.querySelector('span[class=badge]').innerText = data.Counts[el.getAttribute('id')];
  });

  clearTimeout(loading);
  statusDiv.setAttribute('class', 'alert');
  statusDiv.innerText = 'Load complete';
  setTimeout(() => {
    statusDiv.dataset.faded = true;
  }, 5000);
  if (data.Issues == null || data.Issues.length == 0) {
    table.setAttribute('hidden', true);
    emptyDiv.removeAttribute('hidden');
    return
  }

  let total = data.Counts[tab.getAttribute('id')];
  let count = data.Issues.length;
  if (count != total) {
    table.caption.innerText = table.dataset.caption + ` (showing ${count} of ${total})`;
  }
  else {
    table.caption.innerText = table.dataset.caption;
  }
  populateTable(table, data.Issues);
  table.removeAttribute('hidden');
  emptyDiv.setAttribute('hidden', true);
}

function populateTable(table, issues) {
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
