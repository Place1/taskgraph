- would be cool to have a feature that exports all outputs
  to a single tarball for caching in CI systems.
  - i.e. export to tarball and import from tarball


---

- watch feature
  - taskgraph run :build --watch
  - tashgraph run :serve --watch

  - if an input to a task or process changes then it is re-executed
  - any dependent tasks/process are re-executed
  - the tasks dependencies are re-executed as well
