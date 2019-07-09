// ProgressBar maintains the state of the UI elements needed to represent a
// Bootstrap progress bar as used by the NCA uploader
class ProgressBar {
  constructor(fileInfoMessage) {
    this.fileInfoMessage = fileInfoMessage;
  }

  makeUI(parent) {
    this.row = document.createElement("tr");
    this.row.classList.add("upload");
    this.row.classList.add("in-progress");
    parent.appendChild(this.row);

    this.fileInfo = document.createElement("td");
    this.fileInfo.setAttribute("scope", "row");
    this.fileInfo.classList.add("fileinfo");
    this.fileInfo.innerHTML = this.fileInfoMessage;
    this.row.appendChild(this.fileInfo);

    this.progressWrapper = document.createElement("td");
    this.row.appendChild(this.progressWrapper);

    this.progressDiv = document.createElement("div");
    this.progressDiv.classList.add("progress");
    this.progressWrapper.appendChild(this.progressDiv);

    this.actions = document.createElement("td");
    this.actions.classList.add("actions");
    this.row.appendChild(this.actions);

    this.bar = document.createElement("div");
    this.bar.classList.add("progress-bar");
    this.bar.classList.add("progress-bar-striped");
    this.bar.classList.add("progress-bar-animated");
    this.bar.setAttribute("role", "progressbar");
    this.bar.setAttribute("aria-valuemax", 100);
    this.progressDiv.appendChild(this.bar);
  }

  // action removes the previous action button, if any, and adds a new one with
  // the given label, classname, and on-click handler
  action(label, classname, onclick) {
    this.clearAction();
    this.button = document.createElement("button");
    this.button.innerHTML = label
    this.button.classList.add("btn")
    this.button.classList.add(classname)
    this.button.addEventListener("click", onclick);
    this.actions.appendChild(this.button);
  }

  clearAction() {
    if (this.button != null) {
      this.button.remove();
    }
  }

  setValue(pct) {
    this.bar.setAttribute("aria-valuemin", pct);
    this.bar.setAttribute("aria-valuenow", pct);
    this.bar.setAttribute("style", "width: " + pct + "%");
  }

  done() {
    this.setValue(100);
    this.progressDiv.classList.remove("progress");
    this.progressDiv.innerHTML = "Successfully uploaded";
    this.clearAction()
  }

  error(msg) {
    this.row.classList.remove("in-progress");
    this.progressDiv.classList.remove("progress");
    this.progressDiv.innerHTML = "<strong>Error, unable to upload</strong>: " + msg;
    this.row.classList.add("errored");
    this.row.classList.add("skipping");
    this.clearAction()
  }

  skip(msg) {
    this.row.classList.remove("in-progress");
    this.progressDiv.classList.remove("progress");
    this.progressDiv.innerHTML = "<strong>Skipping: </strong>" + msg;
    this.row.classList.add("skipping");
    this.clearAction()
  }

  abort(msg) {
    this.row.classList.remove("in-progress");
    this.progressDiv.classList.remove("progress");
    this.progressDiv.innerHTML = msg;
    this.row.classList.add("aborted");
    this.row.classList.add("skipping");
    this.clearAction()
  }
}
