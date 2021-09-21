// Disable or enable the custom date range fields depending on the state of the
// "Preset Dates" field
window.addEventListener('load', function (event) {
  var dropdown = document.getElementById('preset-date');
  dropdown.addEventListener('change', checkCustomDateEnabled);

  var fakeEvent = {"target": dropdown};
  checkCustomDateEnabled(fakeEvent);
}, false);

function checkCustomDateEnabled(event) {
  var div = document.querySelectorAll('.custom-date').item(0);
   div.querySelectorAll('input').forEach((el) => {
    if (event.target.value == 'custom') {
      div.classList.remove('hidden');
      el.removeAttribute('disabled');
    } else {
      div.classList.add('hidden');
      el.setAttribute('disabled', '');
    }
  });
}
