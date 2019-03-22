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

function fileErrors(file) {
  if (file.size > 100 * 1024 * 1024) {
    return "too big (files cannot be over 100 megs)";
  }
  if (file.type != "application/pdf") {
    return "this file is not a PDF";
  }
  if (haveFile(file)) {
    return "this file is already in the queue";
  }

  return null;
}

var uploadList = [];

function haveFile(file) {
  for (var i = 0; i < uploadList.length; i++) {
    var uf = uploadList[i];
    if (uf.name == file.name && uf.size == file.size) {
      return true;
    }
  }

  return false;
}

function processPDF(file) {
  const row = document.createElement("div");
  row.classList.add("row");
  row.classList.add("upload");
  fileList.appendChild(row);

  const fileDiv = document.createElement("div");
  fileDiv.classList.add("col-sm-3");
  fileDiv.classList.add("fileinfo");
  fileDiv.innerHTML = file.name;
  row.appendChild(fileDiv);

  const progressWrapper = document.createElement("div");
  progressWrapper.classList.add("col-sm-6");
  row.appendChild(progressWrapper);
  const progressDiv = document.createElement("div");
  progressWrapper.appendChild(progressDiv);

  const actionsDiv = document.createElement("div");
  actionsDiv.classList.add("col-sm-3");
  actionsDiv.classList.add("actions");
  row.appendChild(actionsDiv);

  var err = fileErrors(file);
  if (err != null) {
    progressDiv.innerHTML = "Skipping " + file.name + ": " + err;
    row.classList.add("skipping");
    return;
  }

  row.classList.add("in-progress");
  progressDiv.classList.add("progress");

  startUpload(file, progressDiv, actionsDiv);
}

function startUpload(file, progressDiv, actionsDiv) {
  uploadList.push(file);
  const reader = new FileReader();
  const xhr = new XMLHttpRequest();
  this.xhr = xhr;

  const bar = document.createElement("div");
  bar.classList.add("progress-bar");
  bar.classList.add("progress-bar-striped");
  bar.classList.add("progress-bar-animated");
  bar.setAttribute("role", "progressbar");
  bar.setAttribute("aria-valuemin", 0);
  bar.setAttribute("aria-valuemax", 100);
  bar.setAttribute("aria-valuenow", 0);
  bar.setAttribute("style", "width: 0%");
  progressDiv.appendChild(bar);
  this.progressBar = bar

  const cancel = document.createElement("button");
  cancel.classList.add("btn");
  cancel.classList.add("btn-danger");
  cancel.innerHTML = "Cancel";
  actionsDiv.appendChild(cancel);
  this.cancelButton = cancel;
  this.actionsDiv = actionsDiv;

  const self = this;
  this.xhr.upload.addEventListener("progress", function(e) {
    if (e.lengthComputable) {
      const percentage = Math.round((e.loaded * 100) / e.total);
      self.progressBar.setAttribute("style", "width: " + percentage + "%");
      self.progressBar.setAttribute("aria-valuenow", percentage);
      console.log(e)
    }
  }, false);

  xhr.upload.addEventListener("load", function(e) {
    self.progressBar.setAttribute("style", "width: 100%");
    self.progressBar.setAttribute("aria-valuenow", 100);
    self.progressBar.classList.remove("progress-bar-striped");
    self.progressBar.classList.remove("progress-bar-animated");
    console.log(e)
  }, false);

  const form = document.getElementById("uploadform");
  xhr.open("POST", form.action+"/ajax");
  xhr.overrideMimeType('text/plain; charset=x-user-defined-binary');

  reader.onload = function(e) {
    xhr.send(e.target.result);
  };

  reader.readAsBinaryString(file);
}

});
