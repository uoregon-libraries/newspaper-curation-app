function preventDoubleClick(e) {
  const el = e.target;

  // Check if the element is already processing
  if (el.dataset.processing == true) {
    e.preventDefault();
    e.stopPropagation();
    return;
  }

  // Set the element as processing and make it look disabled
  el.dataset.processing = true;
  setTimeout(() => { el.dataset.processing = null; }, 8000);
}

document.addEventListener('DOMContentLoaded', () => {
  // Prevent double-clicking explicitly flagged elements of any kind as well as
  // all submit buttons that are on a "real" form (action is set)
  const elements = document.querySelectorAll('.prevent-double-click, form[action] button[type="submit"]');
  elements.forEach(el => {
    el.addEventListener('click', preventDoubleClick);
  });
});
