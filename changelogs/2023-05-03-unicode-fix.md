### Fixed

- Invalid unicode characters (anything defined as a control rune, private-use
  rune, or "surrogate" rune) are stripped from the output `pdftotext` gives us
  just prior to generating ALTO XML. This prevents MySQL and MariaDB errors
  when ingesting into ONI.
