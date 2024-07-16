document.addEventListener('DOMContentLoaded', () => {
  var lists = document.querySelectorAll(".action-list");
  for (i = 0; i < lists.length; i++) {
    lists[i].scrollTop = lists[i].scrollHeight;
  }
});
