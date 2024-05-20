### Changed

- After curation, a `.manifest` file is generated in the issue's directory
  which contains the SHA256 sums of all files in the issue. This data is then
  used when batches are generated to significantly reduce the time it takes to
  finish generating a batch.
