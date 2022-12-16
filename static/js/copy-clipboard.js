window.addEventListener('load', function (event) {
  let activeClass = "--copy";

  document.querySelectorAll(`button.copy:not(.${activeClass})`).forEach(copyBtn => {
    copyBtn.dataset.originalText = "Copy";

    // Make sure to add this class so we don't double-add the copy button
    copyBtn.classList.add(activeClass);

    copyBtn.addEventListener("click", () => {
      let text = copyBtn.parentElement.querySelector("code").innerText;
      navigator.clipboard.writeText(text).then(
        // Success
        () => {
          copyBtn.innerText = "Copied";
          setTimeout(() => copyBtn.innerText = copyBtn.dataset.originalText, 2000);
        },

        // Fail
        () => {
          copyBtn.innerText = "Unable to copy";
          setTimeout(() => copyBtn.innerText = copyBtn.dataset.originalText, 2000);
        },
      );
    });
  });
});
