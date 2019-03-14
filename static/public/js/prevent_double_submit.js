window.onload = function() {
  var controls = document.querySelectorAll(".prevent-double-click");
  for (x = 0; x < controls.length; x++) {
    preventDblClick(controls[x]);
  }
}

function preventDblClick(control) {
  control.addEventListener("click", function(event) {
    var lastClicked = control.dataset.lastClicked;
    if (lastClicked && Date.now() - lastClicked < 8000) {
      console.log("Skipping click");
      event.preventDefault();
      event.stopPropagation();
      return;
    }
    console.log("Processing click");
    return control.dataset.lastClicked = Date.now();
  });
};
