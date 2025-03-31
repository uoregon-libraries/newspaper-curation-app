(function() {
  'use strict';

  window.addEventListener('DOMContentLoaded', (event) => {
    // If args are present, pre-fill the form and fetch issues
    let u = new URL(window.location);
    let srch = new URLSearchParams(u.search.substr(1));
    document.getElementById('lccn').value = srch.get('lccn');
    document.getElementById('moc').value = srch.get('moc');
    if (srch.get('lccn') || srch.get('moc')) {
      loadIssues();
    }

    // Set up the filter form to fetch JSON from the server on submit
    document.getElementById('filter-form').addEventListener('submit', fetchIssues);
  });

  function fetchIssues(e) {
    e.preventDefault();
    let u = new URL(window.location);
    let srch = new URLSearchParams(u.search.substr(1));
    srch.set('lccn', document.getElementById('lccn').value);
    srch.set('moc', document.getElementById('moc').value);
    u.search = srch.toString();
    history.replaceState(null, "", u);
    loadIssues();
  }

  async function loadIssues() {
    const resultsDiv = document.getElementById('search-results');
    const statusDiv = document.getElementById('json-status');

    // Clear current search results and show loading status
    resultsDiv.innerHTML = '<p><em>Loading...</em></p>';
    statusDiv.setAttribute('class', 'alert alert-info');
    statusDiv.dataset.faded = false;
    statusDiv.innerText = 'Fetching issues from server...';

    let u = new URL(window.location);
    let srch = new URLSearchParams(u.search.substr(1));

    let searchURL = document.getElementById('filter-form').dataset.url;
    var response, data;
    try {
      response = await fetch(searchURL + '?' + srch.toString());
      data = await response.json();
    }
    catch (e) {
      console.error('Exception caught during fetch: ', e);
      statusDiv.setAttribute('class', 'alert alert-danger');
      statusDiv.innerText = 'Network error trying to retrieve issues: please reload the page and try again, or contact support.';
      resultsDiv.innerHTML = '<p><em>Error loading issues.</em></p>';
      return;
    }

    if (!response.ok) {
      statusDiv.setAttribute('class', 'alert alert-warning');
      statusDiv.innerText = data.message;
      resultsDiv.innerHTML = '<p><em>Error loading issues.</em></p>';
      return;
    }

    statusDiv.setAttribute('class', 'alert alert-success');
    statusDiv.innerText = `Load complete: ${data.TotalResults} issues found.`;
    setTimeout(() => {
      statusDiv.dataset.faded = true;
    }, 5000);

    populateSearchResults(resultsDiv, data.Issues);
  }

  function populateSearchResults(container, issues) {
    if (issues == null || issues.length === 0) {
      container.innerHTML = '<p><em>No issues found matching your criteria.</em></p>';
      return;
    }

    container.innerHTML = '';
    const list = document.createElement('ul');
    list.classList.add('list-group');
    container.appendChild(list);

    for (const issue of issues) {
      const listItem = document.createElement('li');
      listItem.classList.add('list-group-item');
      listItem.dataset.issueId = issue.ID;

      const checkbox = document.createElement('input');
      checkbox.type = 'checkbox';
      checkbox.classList.add('form-check-input', 'me-1');
      checkbox.value = issue.ID;
      checkbox.id = `issue-${issue.ID}`;

      const label = document.createElement('label');
      label.htmlFor = `issue-${issue.ID}`;
      label.textContent = ` ${issue.TitleName} (${issue.LCCN}) - ${issue.Date} (Batch: ${issue.BatchName})`;

      listItem.appendChild(checkbox);
      listItem.appendChild(label);
      list.appendChild(listItem);
    }
  }
})();
