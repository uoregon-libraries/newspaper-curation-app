/*
*   This content is licensed according to the W3C Software License at
*   https://www.w3.org/Consortium/Legal/2015/copyright-software-and-document
*/
(function () {
  var tablist = document.querySelectorAll('[role="tablist"]')[0];
  if (tablist == null) {
    return;
  }
  var tabs;
  var panels;
  var delay = determineDelay();

  generateArrays();

  function generateArrays () {
    tabs = document.querySelectorAll('[role="tab"]');
    panels = document.querySelectorAll('[role="tabpanel"]');
  };

  // For easy reference
  var keys = {
    end: 35,
    home: 36,
    left: 37,
    up: 38,
    right: 39,
    down: 40,
    delete: 46
  };

  // Add or substract depenign on key pressed
  var direction = {
    37: -1,
    38: -1,
    39: 1,
    40: 1
  };

  // Bind listeners
  for (i = 0; i < tabs.length; ++i) {
    addListeners(i);
  };

  // Set the active tab
  parseTabState();

  // Make sure that any history change loads the right tab, not just when the
  // client requests a new page from the server
  window.addEventListener('hashchange', parseTabState, false);

  function addListeners (index) {
    tabs[index].addEventListener('click', clickEventListener);
    tabs[index].addEventListener('keydown', keydownEventListener);
    tabs[index].addEventListener('keyup', keyupEventListener);

    // Build an array with all tabs (<button>s) in it
    tabs[index].index = index;
  };

  // When a tab is clicked, activateTab is fired to activate it
  function clickEventListener (event) {
    var tab = event.target;
    // Allow a child of the button to be clicked - for some reason even though
    // the event is defined solely on the button, a span within the button
    // "steals" the event without this.
    if (tab.getAttribute('role') != 'tab') {
      tab.parentNode.dispatchEvent(new Event('click'));
      return;
    }
    activateTab(tab, false);
  };

  // Handle keydown on tabs
  function keydownEventListener (event) {
    if (event.altKey) { return; }
    var key = event.keyCode;

    switch (key) {
      case keys.end:
        event.preventDefault();
        // Activate last tab
        activateTab(tabs[tabs.length - 1]);
        break;
      case keys.home:
        event.preventDefault();
        // Activate first tab
        activateTab(tabs[0]);
        break;

      // Up and down are in keydown
      // because we need to prevent page scroll >:)
      case keys.up:
      case keys.down:
        determineOrientation(event);
        break;
    };
  };

  // Handle keyup on tabs
  function keyupEventListener (event) {
    if (event.altKey) { return; }
    var key = event.keyCode;

    switch (key) {
      case keys.left:
      case keys.right:
        determineOrientation(event);
        break;
      case keys.delete:
        determineDeletable(event);
        break;
    };
  };

  // When a tablist’s aria-orientation is set to vertical,
  // only up and down arrow should function.
  // In all other cases only left and right arrow function.
  function determineOrientation (event) {
    if (event.altKey) { return; }
    var key = event.keyCode;
    var vertical = tablist.getAttribute('aria-orientation') == 'vertical';
    var proceed = false;

    if (vertical) {
      if (key === keys.up || key === keys.down) {
        event.preventDefault();
        proceed = true;
      };
    }
    else {
      if (key === keys.left || key === keys.right) {
        proceed = true;
      };
    };

    if (proceed) {
      switchTabOnArrowPress(event);
    };
  };

  // Either focus the next, previous, first, or last tab
  // depening on key pressed
  function switchTabOnArrowPress (event) {
    if (event.altKey) { return; }
    var pressed = event.keyCode;

    for (x = 0; x < tabs.length; x++) {
      tabs[x].addEventListener('focus', focusEventHandler);
    };

    if (direction[pressed]) {
      var target = event.target;
      if (target.index !== undefined) {
        if (tabs[target.index + direction[pressed]]) {
          tabs[target.index + direction[pressed]].focus();
        }
        else if (pressed === keys.left || pressed === keys.up) {
          focusLastTab();
        }
        else if (pressed === keys.right || pressed == keys.down) {
          focusFirstTab();
        };
      };
    };
  };

  // Returns the tab as defined in the URL fragment ("hash" to javascript)
  function getFragmentTab() {
    var hsh = new URLSearchParams(window.location.hash.substr(1));
    return hsh.get('tab');
  }

  function setFragmentTab(tabid) {
    var hsh = new URLSearchParams(window.location.hash.substr(1));
    hsh.set('tab', tabid);
    window.location.hash = hsh.toString();
  }

  // Activates any given tab panel
  function activateTab (tab, setFocus) {
    setFocus = setFocus || true;
    // Deactivate all other tabs
    deactivateTabs();

    // Remove tabindex attribute
    tab.removeAttribute('tabindex');

    // Set the tab as selected
    tab.setAttribute('aria-selected', 'true');

    // Get the value of aria-controls (which is an ID)
    var controls = tab.getAttribute('aria-controls');

    // Remove hidden attribute from tab panel to make it visible
    document.getElementById(controls).removeAttribute('hidden');

    // Update the URL fragment if it's not already set properly
    if (tab.id != getFragmentTab()) {
      setFragmentTab(tab.id);
    }

    // Set focus when required
    if (setFocus) {
      tab.focus();
    };

    // Fire off an event so tabs can handle custom on-select behaviors
    const ev = new Event('tabselect');
    tab.dispatchEvent(ev);
  };

  // Deactivate all tabs and tab panels
  function deactivateTabs () {
    for (t = 0; t < tabs.length; t++) {
      tabs[t].setAttribute('tabindex', '-1');
      tabs[t].setAttribute('aria-selected', 'false');
      tabs[t].removeEventListener('focus', focusEventHandler);
    };

    for (p = 0; p < panels.length; p++) {
      panels[p].setAttribute('hidden', 'hidden');
    };
  };

  // Make a guess
  function focusFirstTab () {
    tabs[0].focus();
  };

  // Make a guess
  function focusLastTab () {
    tabs[tabs.length - 1].focus();
  };

  // Detect if a tab is deletable
  function determineDeletable (event) {
    target = event.target;

    if (target.getAttribute('data-deletable') !== null) {
      // Delete target tab
      deleteTab(event, target);

      // Update arrays related to tabs widget
      generateArrays();

      // Activate the closest tab to the one that was just deleted
      if (target.index - 1 < 0) {
        activateTab(tabs[0]);
      }
      else {
        activateTab(tabs[target.index - 1]);
      };
    };
  };

  // Deletes a tab and its panel
  function deleteTab (event) {
    var target = event.target;
    var panel = document.getElementById(target.getAttribute('aria-controls'));

    target.parentElement.removeChild(target);
    panel.parentElement.removeChild(panel);
  };

  // Determine whether there should be a delay
  // when user navigates with the arrow keys
  function determineDelay () {
    var hasDelay = tablist.hasAttribute('data-delay');
    var delay = 0;

    if (hasDelay) {
      var delayValue = tablist.getAttribute('data-delay');
      if (delayValue) {
        delay = delayValue;
      }
      else {
        // If no value is specified, default to 300ms
        delay = 300;
      };
    };

    return delay;
  };

  //
  function focusEventHandler (event) {
    var target = event.target;

    setTimeout(checkTabFocus, delay, target);
  };

  // Only activate tab on focus if it still has focus after the delay
  function checkTabFocus (target) {
    focused = document.activeElement;

    if (target === focused) {
      activateTab(target, false);
    };
  };

  // parseTabState determines which tab should be active based on the fragment
  function parseTabState() {
    var tab = document.getElementById(getFragmentTab());
    if (tab == null) {
      tab = tabs[0];
    }
    activateTab(tab);
  }
}());
