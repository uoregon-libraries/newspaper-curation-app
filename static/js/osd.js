var osd;
var fwb = null;

function buildOSD(staticRoot) {
  osd = OpenSeadragon({
    id: "osd-body",

    prefixUrl: staticRoot + "/openseadragon/images/",

    // Keep same view across pages
    preserveViewport: true,

    // Make default view better
    toolbar: "osd-toolbar",
    visibilityRatio: 1.0,
    zoomPerSecond: 0.25,
    immediateRender: true,
    showHomeControl: false,
    showFullPageControl: false,

    // Allow access to all pages
    sequenceMode: true,
    tileSources: tileSources,
  });

  osd.addHandler("page", function(data) {
    $("#osd-image-number").text(data.page + 1)
  });

  // Jump to the top of the page by default
  osd.addHandler("open", osdOpenOnce);
}

function destroyOSD() {
  osd.destroy();
  osd = null;
}

function getFwb() {
  if (fwb == null) {
    osd.viewport.fitHorizontally(true);
    fwb = osd.viewport.getBounds();
  }

  return new OpenSeadragon.Rect(fwb.x, fwb.y, fwb.width, fwb.height);
}

function jumpToTop() {
  var b = getFwb();
  b.y = 0;
  osd.viewport.fitBoundsWithConstraints(b, true);
}

function jumpToBottom() {
  var b = getFwb();
  b.y = 1000;
  osd.viewport.fitBoundsWithConstraints(b, true);
}

function osdOpenOnce() {
  jumpToTop();
  osd.removeHandler("open", osdOpenOnce);
}

$(document).ready(function(){
  $("#osd-jump-top").click(function() {
    jumpToTop();
  });

  $("#osd-jump-bottom").click(function() {
    jumpToBottom();
  });
});
