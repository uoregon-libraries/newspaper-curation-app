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
  constructor(file, fileList) {
    this.file = file;
    this.progress = new ProgressBar(this.file.name);
    this.fileList = fileList;
  }

  start() {
    this.progress.makeUI(this.fileList);
    this.progress.setValue(0);
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

  skip(err) {
    this.progress.skip(err);
  }

  startUpload() {
    // alias self for all the callbacks that would shadow "this"
    const self = this;

    var fileUploader = new Object();
    this.xhr = new XMLHttpRequest();

    this.progress.action("Cancel", "btn-danger", function() {
      self.xhr.upload.userCanceled = true;
      self.xhr.abort();
    });

    var up = this.xhr.upload;
    up.addEventListener("progress", function(e) {
      if (e.lengthComputable) {
        const percentage = Math.round((e.loaded * 100) / e.total);
        self.progress.setValue(percentage);
      }
    }, false);

    this.xhr.addEventListener('readystatechange', function(e) {
      if (this.readyState != 4) {
        return;
      }

      // Don't modify state on abort
      if (this.upload.userCanceled) {
        return;
      }

      if (this.status == 200) {
        self.progress.done();
        uploadQueue.push(this.file);
        console.log("TODO: Add delete button");
        return;
      }

      self.progress.error(this.response);
    });

    up.addEventListener("abort", function(e) {
      self.progress.abort("Canceled");
    }, false);

    up.addEventListener("error", function(e) {
      self.progress.error("Network failure");
    }, false);

    const form = document.getElementById("uploadform");
    const fd = new FormData();
    fd.append("uid", form.elements["uid"].value);
    this.xhr.open("POST", form.action+"/ajax", true);
    fd.append("myfile", this.file);
    this.xhr.send(fd);
  }
}
