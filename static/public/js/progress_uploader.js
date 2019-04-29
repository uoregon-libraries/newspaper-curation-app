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

class ProgressUploader {
  constructor(file) {
    this.file = file;
    this.ui = new Object();
  }

  start() {
    var err = this.fileErrors();
    if (err != null) {
      this.skip(err);
      return;
    }

    uploadList.push(this.file);
    this.startUpload();
  }

  fileErrors() {
    if (this.file.size > 100 * 1024 * 1024) {
      return "too big (files cannot be over 100 megs)";
    }
    if (this.file.type != "application/pdf") {
      return "this file is not a PDF";
    }
    if (haveFile(this.file)) {
      return "this file is already in the queue";
    }

    return null;
  }

  setupUI(el) {
    this.ui.row = document.createElement("div");
    this.ui.row.classList.add("row");
    this.ui.row.classList.add("upload");
    el.appendChild(this.ui.row);

    this.ui.fileDiv = document.createElement("div");
    this.ui.fileDiv.classList.add("col-sm-3");
    this.ui.fileDiv.classList.add("fileinfo");
    this.ui.fileDiv.innerHTML = this.file.name;
    this.ui.row.appendChild(this.ui.fileDiv);

    this.ui.progressWrapper = document.createElement("div");
    this.ui.progressWrapper.classList.add("col-sm-6");
    this.ui.row.appendChild(this.ui.progressWrapper);
    this.ui.progressDiv = document.createElement("div");
    this.ui.progressWrapper.appendChild(this.ui.progressDiv);

    this.ui.actionsDiv = document.createElement("div");
    this.ui.actionsDiv.classList.add("col-sm-3");
    this.ui.actionsDiv.classList.add("actions");
    this.ui.row.appendChild(this.ui.actionsDiv);

    this.ui.row.classList.add("in-progress");
    this.ui.progressDiv.classList.add("progress");

    this.ui.bar = document.createElement("div");
    this.ui.bar.classList.add("progress-bar");
    this.ui.bar.classList.add("progress-bar-striped");
    this.ui.bar.classList.add("progress-bar-animated");
    this.ui.bar.setAttribute("role", "progressbar");
    this.ui.bar.setAttribute("aria-valuemin", 0);
    this.ui.bar.setAttribute("aria-valuemax", 100);
    this.ui.bar.setAttribute("aria-valuenow", 0);
    this.ui.bar.setAttribute("style", "width: 0%");

    this.ui.cancel = document.createElement("button");
    this.ui.cancel.classList.add("btn");
    this.ui.cancel.classList.add("btn-danger");
    this.ui.cancel.innerHTML = "Cancel";
  }

  skip(err) {
    this.ui.row.classList.remove("in-progress");
    this.ui.progressDiv.classList.remove("progress");

    this.ui.progressDiv.innerHTML = "Skipping " + this.file.name + ": " + err;
    this.ui.row.classList.add("skipping");
  }

  startUpload() {
    // alias self for all the callbacks that would shadow "this"
    const self = this;

    var fileUploader = new Object();
    this.xhr = new XMLHttpRequest();

    this.ui.cancel.onclick = function() {
      self.xhr.abort();
    }

    // Attach progress-related components to their respective divs
    this.ui.actionsDiv.appendChild(this.ui.cancel);
    this.ui.progressDiv.appendChild(this.ui.bar);

    this.xhr.upload.addEventListener("progress", function(e) {
      if (e.lengthComputable) {
        const percentage = Math.round((e.loaded * 100) / e.total);
        self.ui.bar.setAttribute("style", "width: " + percentage + "%");
        self.ui.bar.setAttribute("aria-valuenow", percentage);
      }
    }, false);

    this.xhr.upload.addEventListener("load", function(e) {
      self.ui.bar.setAttribute("style", "width: 100%");
      self.ui.bar.setAttribute("aria-valuenow", 100);
      self.ui.bar.classList.remove("progress-bar-striped");
      self.ui.bar.classList.remove("progress-bar-animated");
    }, false);

    const form = document.getElementById("uploadform");
    const fd = new FormData();
    fd.append("uid", form.elements["uid"].value);
    this.xhr.open("POST", form.action+"/ajax", true);
    this.xhr.onreadystatechange = function() {
      console.log("Changed state for uploader " + self.file.name + ": " + self.xhr.readyState + ", status " + self.xhr.status);
    };
    fd.append("myfile", this.file);
    this.xhr.send(fd);
  }
}
