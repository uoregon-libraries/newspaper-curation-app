### Fixed

- Entwined jobs always retry together, even when one fails fatally and is
  manually restarted (via `run-jobs requeue`)
