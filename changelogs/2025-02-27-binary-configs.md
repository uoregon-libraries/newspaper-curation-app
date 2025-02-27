### Changed

- External tools' paths are now configurable. This will enable more people to
  use NCA than before, because custom install paths for some tools could
  prevent NCA from working. For example, it was assuming that `gm` would always
  run the Graphics Magick tools; in a handful of situations this would fail.

### Migration

- Make sure you look at the changes in `settings-example` compared to your
  `settings` file. In particular, you'll need to make sure `GRAPHICS_MAGICK`,
  `PDF_SEPARATE`, and `PDF_TO_TEXT` are set. The defaults will work for most
  people. And if you're already using NCA, the defaults will give you the same
  processing that was already happening.
