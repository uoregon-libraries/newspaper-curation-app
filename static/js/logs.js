// Disable or enable the custom date range fields depending on the state of the
// "Preset Dates" field
window.addEventListener('load', function (event) {
  var dropdown = document.getElementById('preset-date');
  dropdown.addEventListener('change', checkCustomDateEnabled);

  var fakeEvent = {"target": dropdown};
  checkCustomDateEnabled(fakeEvent);
}, false);

function checkCustomDateEnabled(event) {
  document.querySelectorAll('.custom-date-disclosure').forEach((el) => {
    if (event.target.value == 'custom') {
      el.classList.remove('d-none');
      el.querySelectorAll('input').forEach((input) => {
        input.removeAttribute('disabled');
      });
    } else {
      el.classList.add('d-none');
      el.querySelectorAll('input').forEach((input) => {
        input.setAttribute('disabled', '');
      });
    }
  });
}
