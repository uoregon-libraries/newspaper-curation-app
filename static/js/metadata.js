function goToNextUnlabeledPage() {
  const currentPage = parseInt(document.getElementById('osd-image-number').textContent) - 1;
  for (let x = currentPage; x < totalPages; x++) {
    if (pageLabels[x] == null || pageLabels[x] == "") {
      osd.goToPage(x);
      return;
    }
  }

  const x = currentPage + 1;
  osd.goToPage(x % totalPages);
}

document.addEventListener('DOMContentLoaded', function() {
  const pageLabelForm = document.getElementById('page-label-form');
  const metadataForm = document.getElementById('metadata-form');
  const pageLabel = document.getElementById('page-label');
  const pageLabelsCSV = document.getElementById('page-labels-csv');
  const saveDraft = document.getElementById('savedraft');
  const saveQueue = document.getElementById('savequeue');

  if (pageLabelForm != null) {
    pageLabelForm.addEventListener('submit', function(e) {
      const currentPage = parseInt(document.getElementById('osd-image-number').textContent) - 1;
      pageLabels[currentPage] = pageLabel.value;
      pageLabelsCSV.value = pageLabels.join('␟');

      goToNextUnlabeledPage();

      // Try to do an auto-save in the background so users don't lose data if
      // they close the browser or something
      const formData = new FormData(metadataForm);
      formData.append('action', 'autosave');
      fetch(metadataForm.getAttribute('action'), {method: 'POST', body: formData});

      // Don't actually submit the page-label form
      e.preventDefault();
      return false;
    });

    // Make sure page labels are saved on main form submission
    metadataForm.addEventListener('submit', function() {
      const currentPage = parseInt(document.getElementById('osd-image-number').textContent) - 1;
      pageLabels[currentPage] = pageLabel.value;
      pageLabelsCSV.value = pageLabels.join('␟');
    });

    // Don't validate when saving as a draft
    saveDraft.addEventListener('click', function() {
      metadataForm.setAttribute('novalidate', 'novalidate');
    });

    // Make sure validation is reset when queueing for review
    saveQueue.addEventListener('click', function() {
      metadataForm.removeAttribute('novalidate');
    });
  }

  osd.addHandler('page', function(data) {
    if (pageLabel != null) {
      if (pageLabels[data.page] == null) {
        pageLabels[data.page] = "";
      }
      pageLabel.value = pageLabels[data.page];
    }

    const pageLabelText = document.getElementById('page-label-text');
    if (pageLabelText != null) {
      pageLabelText.textContent = pageLabels[data.page];
    }
  });

  osd.goToPage(0);

  if (pageLabel != null) {
    pageLabel.value = pageLabels[0];
  }
});
