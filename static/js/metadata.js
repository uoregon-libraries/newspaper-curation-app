function goToNextUnlabeledPage() {
  var currentPage = parseInt($("#osd-image-number").text()) - 1;
  for (var x = currentPage; x < totalPages; x++) {
    if (pageLabels[x] == null || pageLabels[x] == "") {
      osd.goToPage(x);
      return;
    }
  }

  var x = currentPage + 1;
  osd.goToPage(x % totalPages);
}

$(document).ready(function() {
  $("#page-label-form").submit(function(e) {
    // Store the page label in the global array
    var currentPage = parseInt($("#osd-image-number").text()) - 1;
    pageLabels[currentPage] = $("#page-label").val();
    $("#page-labels-csv").val(pageLabels.join("␟"));

    goToNextUnlabeledPage();

    // Try to do an auto-save in the background so users don't lose data if
    // they close the browser or something
    var form = $("#metadata-form")
    var data = form.serialize();
    data += "&action=autosave";
    $.post(form.attr("action"), data);

    // Don't actually submit the page-label form
    e.preventDefault();
    return false;
  });

  // Make sure page labels are saved on main form submission
  $("#metadata-form").submit(function(e) {
    var currentPage = parseInt($("#osd-image-number").text()) - 1;
    pageLabels[currentPage] = $("#page-label").val();
    $("#page-labels-csv").val(pageLabels.join("␟"));
  });

  // Don't validate when saving as a draft
  $("#savedraft").on("click", function() {
    $("#metadata-form").attr("novalidate", "novalidate");
  });

  // Make sure validation is reset when queueing for review
  $("#savequeue").on("click", function() {
    $("#metadata-form").removeAttr("novalidate");
  });

  osd.addHandler("page", function(data) {
    $('#page-label').val(pageLabels[data.page]);
    $('#page-label-text').text(pageLabels[data.page]);
  });

  osd.goToPage(0);
  $('#page-label').val(pageLabels[0]);
});
