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

function processPDF(file) {
  var uploader = new ProgressUploader(file, fileList);
  uploader.start();
}

});
