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
// (2) The table summary includes an instruction for screen reader users
//     explaining that the table can be sorted by clicking on table headers
// (3) The sort status (ascending, descending) is indicated using an
//     abbreviation element with a title attribute that can be read by screen
//     readers
// (4) Focus is refreshed whenever sort status is changed, prompting screen
//     readers to read the new information
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
// Default settings can be overriden by passing a settings object to the constructor, e.g.:
//   SortableTable.initAll({ summary: "(Click a column header to sort)", ... })

SortableTable = function(table, settings) {
  /// <summary>Enables tables to be sorted dynamically</summary>
  /// <param name="table" type="DomElement">Table to be made sortable</param>
  /// <param name="settings" type="object" optional="true">Optional settings in object literal notation, e.g., { summary: "(Click a column header to sort)", ... }</param>

  // Configurable settings
  var settings = settings || {};
  this._summary = typeof settings.summary != "undefined" ? settings.summary : "(Click a column header to sort)";
  this._unsortedIcon = typeof settings.unsortedIcon != "undefined" ? settings.unsortedIcon : "\u2195"; // up down arrow
  this._unsortedText = typeof settings.unsortedText != "undefined" ? settings.unsortedText : "";
  this._ascendingIcon = typeof settings.ascendingIcon != "undefined" ? settings.ascendingIcon : "\u2193"; // downwards arrow
  this._ascendingText = typeof settings.ascendingText != "undefined" ? settings.ascendingText : "(sorted ascending)";
  this._descendingIcon = typeof settings.descendingIcon != "undefined" ? settings.descendingIcon : "\u2191"; // upwards arrow
  this._descendingText = typeof settings.descendingText != "undefined" ? settings.descendingText : "(sorted descending)";
  this._numberPattern = typeof settings.numberPattern != "undefined" ? settings.numberPattern : "^\\s*-?\\$?[\\d,]*\\.?\\d*%?$"; // any number of whitespace characters, optional negative sign (hyphen), optional dollar sign, any number of digits/commas, optional period, any number of digits (note: will match all white-space or empty-string)
  this._numberCleanUpPattern = typeof settings.numberCleanUpPattern != "undefined" ? settings.numberCleanUpPattern : "[$,]"; // dollar sign or comma
  this._minDate = typeof settings.minDate != "undefined" && Date.parse(settings.minDate) ? Date.parse(settings.minDate) : Date.parse("1/1/1900");

  // "Constants"
  this._sortButtonClassName = "sort-button";
  this._sortIconClassName = "sort-icon";
  this._unsortedClassName = "unsorted";
  this._ascendingClassName = "ascending";
  this._descendingClassName = "descending";
  this._sortTypeDate = "date";
  this._sortTypeNumber = "number";
  this._sortTypeAlpha = "alpha";

  // class variables
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
    /// <summary>Adds sort buttons and sort icons (abbr elements) to the table headers.</summary>
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
      // create sort button
      var sortButton = document.createElement("button");
      sortButton.className = this._sortButtonClassName;
      sortButton.onclick = createDelegate(this, this.sort, [i]);

      // move contents of header into sort button
      while (th.childNodes.length > 0) {
        sortButton.appendChild(th.childNodes[0]);
      }
      // create sort icon
      var sortIcon = document.createElement("abbr");
      sortIcon.appendChild(document.createTextNode(this._unsortedIcon));
      sortIcon.title = this._unsortedText;
      sortIcon.className = this._sortIconClassName;
      sortIcon.style.borderStyle = "none";
      // append sort button & sort icon
      sortButton.sortIcon = sortButton.appendChild(sortIcon);
      th.sortButton = th.appendChild(sortButton);
    }

    if (hasSortableColumns) {
      // add summary
      if (this._summary.length > 0) {
        this._table.summary += " " + this._summary;
      }
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

        var numberCleanUpRegExp = new RegExp(this._numberCleanUpPattern, "ig"); // non-numeric characters allowed before or within numbers (e.g. dollar sign and comma)
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

    // Reset old column's sort icon
    var oldTH = this._tHeadRow.cells[oldIdx];
    var th = this._tHeadRow.cells[idx];
    if (oldIdx != null && oldIdx != idx) {
      oldTH.classList.remove(this._ascendingClassName);
      oldTH.classList.remove(this._descendingClassName);
      oldTH.classList.add(this._unsortedClassName);
      oldTH.sortButton.sortIcon.innerText = this._unsortedIcon;
      th.classList.remove(this._unsortedClassName);
    }
    else {
      th.classList.remove(this._ascendingClassName);
      th.classList.remove(this._descendingClassName);
    }

    if (this._isAscending) {
      th.classList.add(this._ascendingClassName);
      th.sortButton.sortIcon.innerText = this._ascendingIcon;
    }
    else {
      th.classList.add(this._descendingClassName);
      th.sortButton.sortIcon.innerText = this._descendingIcon;
    }
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

SortableTable.init = function(table, settings) {
  /// <summary>Static method that initializes a single SortableTable.</summary>
  /// <param name="table" type="DomElement">Table to be made sortable</param>
  /// <param name="settings" type="object" optional="true">Optional settings in object literal notation, e.g., { className: "sortable", summary: "(Click a column header to sort)", ... }</param>
  if (document.getElementsByTagName && document.createElement && Function.apply) {
    if (SortableTable.isSortable(table)) {
      var sortableTable = new SortableTable(table, settings);
    }
  }
}

SortableTable.initAll = function(settings) {
  /// <summary>Static method that initializes all SortableTables in a document.</summary>
  /// <param name="settings" type="Object" optional="true">Optional settings in object literal notation, e.g., { summary: "(Click a column header to sort)", ...}</param>
  var tables = document.querySelectorAll("table.sortable");
  for (var i = 0, n = tables.length; i < n; i++) {
    SortableTable.init(tables[i], settings);
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
