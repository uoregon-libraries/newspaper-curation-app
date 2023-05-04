### Fixed

- Derivative-generating jobs now fail after only 4 retries (5 tries total)
  instead of 25 (26 total). Failures with these jobs are almost always fatal,
  and we want them out of NCA sooner in order to fix the underlying problems
  manually (e.g., a corrupt PDF).
