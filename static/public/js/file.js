document.addEventListener("DOMContentLoaded", function() {

window.URL = window.URL || window.webkitURL;

const fileSelect = document.getElementById("file-select");
const fileUpload = document.getElementById("file-upload");
fileSelect.addEventListener("click", function (e) {
  if (fileUpload) {
    fileUpload.click();
  }
}, false);
fileUpload.addEventListener("change", handleFiles, false);

const folderSelect = document.getElementById("folder-select");
const folderUpload = document.getElementById("folder-upload");
folderSelect.addEventListener("click", function (e) {
  if (folderUpload) {
    folderUpload.click();
  }
}, false);
folderUpload.addEventListener("change", handleFiles, false);

const fileList = document.getElementById("file-list");
function handleFiles() {
  const files = this.files;
  for (let i = 0; i < files.length; i++) {
    processPDF(files[i]);
  }
}

// Set up completed progress bar elements for files that were already loaded
for (var x = 0; x < loadedFiles.length; x++) {
  var pb = new ProgressBar(loadedFiles[x].Name);
  pb.makeUI(fileList);
  pb.done();
}

function processPDF(file) {
  var uploader = new ProgressUploader(file, fileList);
  uploader.start();
}

});
