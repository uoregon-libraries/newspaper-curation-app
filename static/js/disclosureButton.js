/*
*   This content is licensed according to the W3C Software License at
*   https://www.w3.org/Consortium/Legal/2015/copyright-software-and-document
*
*   File:   ButtonExpand.js
*
*   Desc:   Disclosure button widget that implements ARIA Authoring Practices
*/

var ButtonExpand = function (domNode) {
  this.domNode = domNode;
  this.keyCode = Object.freeze({
    'RETURN': 13
  });
};

ButtonExpand.prototype.init = function () {
  this.controlledNode = false;
  var id = this.domNode.getAttribute('aria-controls');
  if (id) {
    this.controlledNode = document.getElementById(id);
  }

  // Check URL parameters to determine initial state. Default to closed.
  var params = new URLSearchParams(window.location.search);
  var startOpen = params.get(id) === 'open';
  if (startOpen) {
    this.domNode.setAttribute('aria-expanded', 'true');
    this.showContent();
  } else {
    this.domNode.setAttribute('aria-expanded', 'false');
    this.hideContent();
  }

  this.domNode.addEventListener('keydown',    this.handleKeydown.bind(this));
  this.domNode.addEventListener('click',      this.handleClick.bind(this));
  this.domNode.addEventListener('focus',      this.handleFocus.bind(this));
  this.domNode.addEventListener('blur',       this.handleBlur.bind(this));
};

ButtonExpand.prototype.showContent = function () {
  if (this.controlledNode) {
    this.controlledNode.style.display = 'block';
  }
};

ButtonExpand.prototype.hideContent = function () {
  if (this.controlledNode) {
    this.controlledNode.style.display = 'none';
  }
};

ButtonExpand.prototype.setUrlParam = function (key, value) {
  var params = new URLSearchParams(window.location.search);
  params.set(key, value);
  var newRelativePathQuery = window.location.pathname + '?' + params.toString();
  history.replaceState(null, '', newRelativePathQuery);
};

ButtonExpand.prototype.toggleExpand = function () {
  var id = this.domNode.getAttribute('aria-controls');

  if (this.domNode.getAttribute('aria-expanded') === 'true') {
    this.domNode.setAttribute('aria-expanded', 'false');
    this.hideContent();
    if (id) {
      this.setUrlParam(id, 'closed');
    }
  }
  else {
    this.domNode.setAttribute('aria-expanded', 'true');
    this.showContent();
    if (id) {
      this.setUrlParam(id, 'open');
    }
  }
};

/* EVENT HANDLERS */

ButtonExpand.prototype.handleKeydown = function (event) {
  console.log('[keydown]');
  switch (event.keyCode) {
    case this.keyCode.RETURN:
      this.toggleExpand();
      event.stopPropagation();
      event.preventDefault();
      break;

    default:
      break;
  }
};

ButtonExpand.prototype.handleClick = function (event) {
  this.toggleExpand();
};

ButtonExpand.prototype.handleFocus = function (event) {
  this.domNode.classList.add('focus');
};

ButtonExpand.prototype.handleBlur = function (event) {
  this.domNode.classList.remove('focus');
};

/* Initialize Hide/Show Buttons */

window.addEventListener('load', function (event) {
  var buttons =  document.querySelectorAll('[data-widget=simple-disclosure]');
  for (var i = 0; i < buttons.length; i++) {
    var be = new ButtonExpand(buttons[i]);
    be.init();
  }
}, false);
