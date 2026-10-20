## vX.Y.Z

### Fixed

- METS XML now uses each page's actual label for its "page number" detail.
  Previously, any page with a label got its *sequence number* written instead,
  so batches generated before this fix will have incorrect page labels in ONI.
