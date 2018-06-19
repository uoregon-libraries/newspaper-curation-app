// Copyright (c) 2008, State of Illinois, Department of Human Services. All rights reserved.
// Developed by: MSF&W Accessibility Solutions, http://www.msfw.com/accessibility
// Subject to University of Illinois/NCSA Open Source License
// See: http://www.dhs.state.il.us/opensource
// Version Date: 2008-07-30
//
// Updated 2018-06-12 by UO Libraries to simplify code for easier maintenance,
// remove some unnecessary magic, and improve accessibility
//
// Accessible Sortable Table
//
// This script makes html tables sortable in a manner that is usable with
// keyboard commands, large fonts, screen readers, and speech recognition
// tools, specifically:
// (1) Sorting is activated using actual buttons, which are focusable and
//     clickable from the keyboard and by assistive technologies
// (2) Adds a div above the table with information about how to sort as well as
//     the current sort, using aria-describedby and aria-live
// (3) The sort status (ascending, descending) is indicated using an
//     element with aria attributes that can be read by screen readers
// (4) When sort status changes, the aria-live region also changes, so users
//     know something happened
//
// To make a table sortable, simply add the class "sortable" to the table, add
// a sort-type data tag to table headers (e.g., data-sorttype="alpha"), and call
// SortableTable.initAll().
//
// The sort type (alphabetical, numeric, date) is determined by setting a data
// attribute ("data-sorttype") on any column header:
//   data-sorttype="alpha" - for case-insensitive alphabetical sorting
//   data-sorttype="number" - for integers, decimals, money ($##.##), and percents (##%)
//   data-sorttype="date" - for "mm/dd/yyyy" and "month dd, yyyy" format dates (use alpha for "yyyy-mm-dd")
//
// A custom sort key (value to use for sorting) can be indicated for any data
// cell by setting a data attribute on the cell:
//   data-sortkey="<value>" - where value is the value to use for sorting
//
// Table head (thead) and footer (tfoot) rows are not sorted.
// If no table head is present, one will be created around the first row.
//
// Details on the precise HTML fiddling which occurs:
//
// - <div class="sortable-info-wrapper"> is added above the table
// - <div class="description"> is added inside this wrapper
//   - This element is referenced by the table via an aria-describedby attribute
//   - This element is used to explain that clicking column headers sort the table
// - <div class="status" aria-live="polite"> is added inside this wrapper
//   - This element is initially blank
// - The table headers which are sortable get wrapped in buttons
// - A screen-reader-hidden unicode "up-down arrow" is appended to each button
// - When a sort button is clicked:
//   - The table rows are sorted (obviously)
//   - The sort status icon changes to an up or down arrow
//   - The live region is updated, e.g., "(sorted by Login, ascending)"
//   - The header's aria-sort attribute is set to ascending or descending
//   - If a prior header was sorted, its arrow is replaced with the up-down unicode arrow
//   - If a prior header was sorted, its aria-sort attribute is removed

SortableTable = function(table) {
  /// <summary>Enables tables to be sorted dynamically</summary>
  /// <param name="table" type="DomElement">Table to be made sortable</param>

  // "Constants"

  // Description associated to the table
  this._desc = "Click a column header to sort.  Click a second time to reverse the sort.";

  // Up/down arrow icon (↕): column is unsorted
  this._unsortedIcon = "\u2195";

  // Downwards arrow icon (↓): column is sorted ascending
  this._ascendingIcon = "\u2193";

  // Upwards arrow icon (↑): column is sorted descending
  this._descendingIcon = "\u2191";

  // Anything matching this is considered a valid number
  this._numberPattern = "^\\s*-?\\$?[\\d,]*\\.?\\d*%?$";

  // Characters allowed in numbers, but stripped prior to parsing
  this._numberCleanUpPattern = "[$,]";

  // Default date assigned to unparseable cells in columns sorted as dates
  this._minDate = Date.parse("1/1/1900");

  // Our three sort types
  this._sortTypeDate = "date";
  this._sortTypeNumber = "number";
  this._sortTypeAlpha = "alpha";

  // Variables defining this object's sort state
  this._table = table;
  this._tBody = this._table.tBodies[0];
  this._tHeadRow = null;
  this._sortedColumnIndex = null;
  this._isAscending = false;

  // initialization
  this.setTHead();
  this.addSortButtons();
}

SortableTable.prototype = {
  setTHead: function() {
    /// <summary>Identifies the head row (the last row in the table head). Creates a thead element if necessary.</summary>
    var tHead = this._table.tHead;
    if (!tHead) {
      tHead = this._table.createTHead();
      tHead.appendChild(this._table.rows[0]);
    }
    this._tHeadRow = tHead.rows[tHead.rows.length - 1];
  },

  addSortButtons: function() {
    /// <summary>Adds sort buttons to the table headers.</summary>
    var hasSortableColumns = false;
    for (var i = 0, n = this._tHeadRow.cells.length; i < n; i++) {
      var th = this._tHeadRow.cells[i];
      // check for sort type class and that header has content
      var st = th.dataset.sorttype;
      if (st != this._sortTypeDate && st != this._sortTypeAlpha && st != this._sortTypeNumber) {
        continue;
      }
      if (th.innerText.length == 0) {
        continue
      }

      hasSortableColumns = true;

      // Create sort button
      var sortButton = document.createElement("button");
      sortButton.classList.add("sort-button");
      sortButton.onclick = createDelegate(this, this.sort, [i]);

      // Move contents of header into sort button
      while (th.childNodes.length > 0) {
        sortButton.appendChild(th.childNodes[0]);
      }

      // Create sort icon for sighted users
      var sortIcon = document.createElement("span");
      sortIcon.classList.add("sort-icon");
      sortIcon.appendChild(document.createTextNode(this._unsortedIcon));
      sortIcon.setAttribute("aria-hidden", "true");

      // append sort button & sort icon
      sortButton.sortIcon = sortButton.appendChild(sortIcon);
      th.sortButton = th.appendChild(sortButton);
    }

    // Add table description and live region if the table was found to be sortable
    if (hasSortableColumns) {
      // Wrapper div
      var info = document.createElement("div");
      info.classList.add("sortable-info-wrapper")
      this._table.parentElement.insertBefore(info, this._table);

      // Inner description div
      var desc = document.createElement("div");
      info.appendChild(desc)
      desc.classList.add("description");
      desc.id = uuidv4();
      desc.innerText = this._desc;
      this._table.setAttribute("aria-describedby", desc.id);

      // The live region for letting people know sorting has changed
      var liveregion = document.createElement("div");
      info.appendChild(liveregion)
      liveregion.classList.add("status");
      liveregion.id = uuidv4();
      liveregion.setAttribute("aria-live", "polite");
      this._table.dataset.sortstatus = liveregion.id;
    }
  },

  sort: function(columnIndex) {
    /// <summary>Sorts the table on the selected column.</summary>
    /// <param name="columnIndex" type="Number">Index of the column on which to sort the table.</param>
    /// <returns type="Boolean">False, to cancel associated click event.</returns>
    var th = this._tHeadRow.cells[columnIndex];
    var rows = this._tBody.rows;
    if (th && rows[0].cells[columnIndex]) {
      var rowArray = [];
      // sort on a new column
      if (columnIndex != this._sortedColumnIndex) {
        // get sort type
        var sortType = th.dataset.sorttype;

        var numberCleanUpRegExp = new RegExp(this._numberCleanUpPattern, "ig");
        for (var i = 0, n = rows.length; i < n; i++) {
          var cell = rows[i].cells[columnIndex];
          var sortKey = cell.dataset.sortkey;
          if (sortKey == null || sortKey == "") {
            sortKey = cell.innerText;
          }

          // convert to date
          if (sortType == this._sortTypeDate) {
            sortKey = Date.parse(sortKey) || this._minDate;
          }
          // convert to number
          else if (sortType == this._sortTypeNumber) {
            sortKey = parseFloat(sortKey.replace(numberCleanUpRegExp, "")) || 0;
          }
          // convert to string (left-trimmed, lowercase)
          else if (sortKey.length > 0) {
            sortKey = sortKey.replace(/^\s+/, "").toLowerCase();
          }
          // add object to rowArray
          rowArray[rowArray.length] = {
            sortKey: sortKey,
            row: rows[i]
          };
        }

        // sort
        rowArray.sort(sortType == this._sortTypeDate || sortType == this._sortTypeNumber ? this.sortNumber : this.sortAlpha);
        this._isAscending = true;
      }
      // sort on previously sorted column
      else {
        // reverse rows (faster than re-sorting)
        for (var i = rows.length - 1; i >= 0; i--) {
          rowArray[rowArray.length] = {
            row: rows[i]
          }
        }
        this._isAscending = !this._isAscending;
      }

      // append rows
      for (var i = 0, n = rowArray.length; i < n; i++) {
        this._tBody.appendChild(rowArray[i].row);
      }

      // clean up
      delete rowArray;

      this.setSortColumn(columnIndex);
    }
    // cancel click event
    return false;
  },

  setSortColumn: function(idx) {
    var oldIdx = this._sortedColumnIndex;
    this._sortedColumnIndex = idx;

    // Reset old column's sort icon and classlist
    var oldTH = this._tHeadRow.cells[oldIdx];
    var th = this._tHeadRow.cells[idx];
    if (oldTH != null) {
      oldTH.removeAttribute("aria-sort");
      if (oldIdx != idx) {
        oldTH.classList.remove("ascending");
        oldTH.classList.remove("descending");
        oldTH.classList.add("unsorted");
        oldTH.sortButton.sortIcon.innerText = this._unsortedIcon;
      }
    }

    // For simplicity, we just remove all sort classes from the new header
    th.classList.remove("unsorted");
    th.classList.remove("ascending");
    th.classList.remove("descending");

    var liveregion = document.getElementById(this._table.dataset.sortstatus);
    // Clear the sort icon so we can use the table header's innerText to label the live region
    th.sortButton.sortIcon.innerText = "";
    var direction = this._isAscending ? "ascending" : "descending";
    liveregion.innerText = "(sorted by " + th.innerText + ", " + direction + ")";
    th.setAttribute("aria-sort", direction);
    th.classList.add(direction);
    th.sortButton.sortIcon.innerText = this._isAscending ? this._ascendingIcon : this._descendingIcon;
  },

  sortNumber: function(a, b) {
    /// <summary>Array sort compare function for number and date columns</summary>
    /// <param name="a" type="Object">rowArray element with number sortKey property</param>
    /// <param name="b" type="Object">rowArray element with number sortKey property</param>
    /// <returns type="Number">Returns a positive number if a.sortKey > b.sortKey, a negative number if a.sortKey < b.sortKey, or 0 if a.sortKey = b.sortKey</returns>
    return a.sortKey - b.sortKey;
  },

  sortAlpha: function(a, b) {
    /// <summary>Array sort compare function for alpha (string) columns</summary>
    /// <param name="a" type="Object">rowArray element with string sortKey property</param>
    /// <param name="b" type="Object">rowArray element with string sortKey property</param>
    /// <returns type="Number">Returns a positive number if a.sortKey > b.sortKey, a negative number if a.sortKey < b.sortKey, or 0 if a.sortKey = b.sortKey</returns>
    return ((a.sortKey < b.sortKey) ? -1 : ((a.sortKey > b.sortKey) ? 1 : 0));
  }
}

SortableTable.init = function(table) {
  /// <summary>Static method that initializes a single SortableTable.</summary>
  /// <param name="table" type="DomElement">Table to be made sortable</param>
  if (document.getElementsByTagName && document.createElement && Function.apply) {
    if (SortableTable.isSortable(table)) {
      var sortableTable = new SortableTable(table);
    }
  }
}

SortableTable.initAll = function() {
  /// <summary>Static method that initializes all SortableTables in a document.</summary>
  var tables = document.querySelectorAll("table.sortable");
  for (var i = 0, n = tables.length; i < n; i++) {
    SortableTable.init(tables[i]);
  }
}

SortableTable.isSortable = function(table) {
  /// <summary>Static method that indicates whether a table can be made sortable (has a single tbody, at least three rows, and a uniform number of columns)</summary>
  /// <param name="table" type="DomElement"></param>
  /// <returns type="Boolean"></returns>
  // check table, single tbody, three rows (including thead)
  if (table == null || table.tBodies.length > 1 || table.rows.length < 3) {
    return false;
  }
  // check uniform columns
  var tBody = table.tBodies[0];
  var numberOfColumns = tBody.rows[0].cells.length;
  for (var i = 0, n = tBody.rows.length; i < n; i++) {
    if (tBody.rows[i].cells.length != numberOfColumns) {
      return false;
    }
  }
  return true;
}

// Creates a delegate to allow the specified method to run in the context of
// the specified instance
function createDelegate(instance, method, argumentsArray) {
  return function() {
    return method.apply(instance, argumentsArray);
  }
}

// Returns a valid v4 uuid... according to stackoverflow, anyway....
//
// See the bottom part of https://stackoverflow.com/a/2117523/468391
function uuidv4() {
  return ([1e7] + -1e3 + -4e3 + -8e3 + -1e11).replace(/[018]/g, function (c) {
    return (c ^ crypto.getRandomValues(new Uint8Array(1))[0] & 15 >> c / 4).toString(16);
  });
}
