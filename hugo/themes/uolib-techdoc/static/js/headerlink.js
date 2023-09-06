(function() {
  document.addEventListener('DOMContentLoaded', function(){
    const targetElements = document.querySelector('main').querySelectorAll('h1, h2, h3, h4, h5, h6');

    targetElements.forEach(( el ) => {
      if (el.id) {
        const headerlink = document.createElement('a');
        headerlink.setAttribute('class', 'headerlink');
        headerlink.setAttribute('href', '#' + el.id);
        headerlink.innerText = "#";
        headerlink.setAttribute('aria-label', 'Permalink to section: ' + el.innerText);

        el.append(headerlink);
      }
    });
  });
})();
