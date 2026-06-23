# resolver — DEPRECATED

> **This package is superseded by [`libs/modplan`](../modplan/).  
> Do not add new code here. Use `libs/modplan` for all mod dependency resolution.**

---

This placeholder was the original resolver abstraction (package dependency resolution, manifest graph processing).  
It has been replaced by the RFC-0031 implementation in `libs/modplan/resolve.go` with full cycle detection, conflict resolution, and load order output.
