var $;

$ = jQuery;

$(function() {
  return $(".prevent-double-click").each(function() {
    return $(this).prevent_double_click();
  });
});

$.fn.prevent_double_click = function() {
  var control;
  control = $(this);
  return control.click(function(event) {
    var last_clicked;
    last_clicked = control.data("last-clicked");
    if (last_clicked && jQuery.now() - last_clicked < 8000) {
      console.log("Skipping click");
      event.preventDefault();
      event.stopPropagation();
      return;
    }
    console.log("Processing click");
    return control.data("last-clicked", jQuery.now());
  });
};
