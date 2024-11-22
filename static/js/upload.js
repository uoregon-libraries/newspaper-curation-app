window.addEventListener('load', function () {
  const input = document.getElementById('marc-uploader');
  const preview = document.getElementById('preview');

  input.addEventListener('change', updateFileList);

  function updateFileList() {
    while(preview.firstChild) {
      preview.removeChild(preview.firstChild);
    }

    const curFiles = input.files;
    if(curFiles.length === 0) {
      const para = document.createElement('p');
      para.textContent = 'No files currently selected for upload';
      preview.appendChild(para);
      return;
    }

    const list = document.createElement('ol');
    preview.appendChild(list);

    for(const file of curFiles) {
      const listItem = document.createElement('li');
      const para = document.createElement('p');

      para.textContent = `File name "${file.name}", file size ${fileSizeHuman(file.size)}.`;
      listItem.appendChild(para);
      list.appendChild(listItem);
    }
  }

  function fileSizeHuman(number) {
    if (number < 1024) {
      return number + 'bytes';
    }
    if (number > 1024 && number < 1048576) {
      return (number/1024).toFixed(1) + 'KB';
    }
    if (number > 1048576) {
      return (number/1048576).toFixed(1) + 'MB';
    }
  }
});
