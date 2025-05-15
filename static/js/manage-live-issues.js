(function() {
  'use strict';

  const filterParams = ['lccn', 'moc', 'went-live', 'url', 'pubdate'];
  let statusFadeTimeout = null;

  window.addEventListener('DOMContentLoaded', (event) => {
    // If args are present, pre-fill the form and fetch issues
    let u = new URL(window.location);
    let srch = new URLSearchParams(u.search.substr(1));
    let filtersPresent = false;
    for (const param of filterParams) {
      const value = srch.get(param);
      document.getElementById(param).value = value;
      if (value) {
        filtersPresent = true;
      }
    }

    if (filtersPresent) {
      loadIssues();
    }

    // Set up the filter form to fetch JSON from the server on submit
    document.getElementById('filter-form').addEventListener('submit', fetchIssues);
  });

  function fetchIssues(e) {
    e.preventDefault();

    // Translate the URL arg (if present), and replace fields before we set the
    // URL query
    let fields = parseProdURL(document.getElementById('url').value);
    if (fields['lccn']) {
      document.getElementById('lccn').value = fields['lccn'];
      document.getElementById('moc').value = '';
      if (fields['date']) {
        document.getElementById('pubdate').value = fields['date'];
        document.getElementById('went-live').value = '';
      }
    }

    let u = new URL(window.location);
    let srch = new URLSearchParams(u.search.substr(1));
    for (const param of filterParams) {
      srch.set(param, document.getElementById(param).value);
    }
    u.search = srch.toString();
    history.replaceState(null, '', u);
    loadIssues();
  }

  // parseProdURL sets form fields depending on the URL value:
  //
  // - If it's empty or invalid, we just ignore it
  // - If valid, overwrite LCCN and erase MOC
  // - If valid and an issue URL, also overwrite date and set went-live to
  //   the initial (anytime) value
  function parseProdURL(val) {
    const url = new URL(val);
    const parts = url.pathname.split('/');
    let result = {};

    // Sanity check: do we have "lccn"? It will be *second* because the URL
    // path will always start with "/"
    if (parts.length < 3 || parts[1] !== 'lccn') {
      return result;
    }

    // Extract segments after "lccn"
    const segments = parts.slice(2);
    result['lccn'] = segments[0];

    // If we have another segment, and it's not blank, it's the issue date
    if (segments.length > 1 && segments[1] != '') {
      result['date'] = segments[1];
    }

    return result;
  }

  async function loadIssues() {
    const srchMessage = document.getElementById('search-results-message');
    const table = document.getElementById('search-results-table');
    const statusDiv = document.getElementById('json-status');

    // Cancel any pending fade-out of the status message
    if (statusFadeTimeout) {
      clearTimeout(statusFadeTimeout);
      statusFadeTimeout = null;
    }

    // Clear current search results and show loading status
    srchMessage.innerHTML = '<p><em>Loading...</em></p>';
    srchMessage.style.display = 'block';
    table.style.display = 'none';

    let loading = setTimeout(() => {
      statusDiv.setAttribute('class', 'inline alert alert-info');
      statusDiv.dataset.faded = false;
      statusDiv.innerText = 'Fetching issues from server...';
    }, 200);

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
      statusDiv.setAttribute('class', 'inline alert alert-danger');
      statusDiv.innerText = 'Network error trying to retrieve issues: please reload the page and try again, or contact support.';
      srchMessage.innerHTML = '<p><em>Error loading issues.</em></p>';
      srchMessage.style.display = 'block';
      table.style.display = 'none';
      return;
    }

    if (!response.ok) {
      statusDiv.setAttribute('class', 'inline alert alert-warning');
      statusDiv.innerText = data.Message;
      srchMessage.innerHTML = '<p><em>Error loading issues.</em></p>';
      srchMessage.style.display = 'block';
      table.style.display = 'none';
      return;
    }

    clearTimeout(loading);
    statusDiv.setAttribute('class', 'inline alert');
    statusDiv.innerText = `Load complete: ${data.TotalResults} issues matched.`;
    statusFadeTimeout = setTimeout(() => {
      statusDiv.dataset.faded = true;
      statusFadeTimeout = null;
    }, 5000);

    populateSearchResults(data.Issues, data.TotalResults);
  }

  function makeLink(href, text) {
    const link = document.createElement('a');
    link.href = href;
    link.textContent = text;

    return link
  }

  function populateSearchResults(issues, totalResults) {
    const srchMessage = document.getElementById('search-results-message');
    const table = document.getElementById('search-results-table');
    const tbody = table.tBodies[0];
    const tcaption = table.caption;

    tbody.innerHTML = '';

    if (issues == null || issues.length === 0) {
      srchMessage.innerHTML = '<p><em>No issues found matching your criteria.</em></p>';
      srchMessage.style.display = 'block';
      table.style.display = 'none';
      return;
    }

    srchMessage.style.display = 'none';
    table.style.display = 'table';

    if (issues.length < totalResults) {
      tcaption.textContent = `Search results: too many matches; showing ${issues.length} (of ${totalResults}).`;
    } else {
      tcaption.textContent = `Search results: ${issues.length} matches`;
    }

    for (const issue of issues) {
      const row = tbody.insertRow();

      row.insertCell().appendChild(makeLink(issue.LiveTitleURL, issue.FullTitle));
      row.insertCell().appendChild(makeLink(issue.LiveIssueURL, issue.PublishedOn + ", ed. " + issue.Edition));
      row.insertCell().appendChild(makeLink(issue.LiveBatchURL, issue.BatchName));
      row.insertCell().textContent = issue.WentLiveAt;

      const cell = row.insertCell();
      const link = document.createElement('a');
      let u = new URL(table.dataset.queueRemovalUrl, window.location.origin);
      u.searchParams.set('id', issue.ID);
      link.href = u.href;
      link.innerHTML = 'Remove...';
      link.setAttribute('class', 'btn btn-danger');
      cell.appendChild(link);
    }
  }
})();
