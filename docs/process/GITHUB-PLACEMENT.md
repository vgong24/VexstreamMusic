# GitHub placement — VexStream Music 3.0.0

```text
repository=vgong24/VexstreamMusic
sourceBaseLane=github.pull.VexstreamMusic.1
mobileSuccessorLane=github.pull.VexstreamMusic.2
matchingConsumer=github.pull.VexLife-Android.13
currentSourceVersion=3.0.0
continuity=[VXG RealForever]
sourceMode=FIRST_CLASS_BUILDABLE_SOURCE
binaryPatchDevelopment=REJECTED
```

PR #1 remains the current 2.2.0 Explore/source baseline. PR #2 is the explicit stacked 3.0.0
VexLife-mobile successor and must not be collapsed back into an unrelated second desktop writer.

The matching Android consumer is VexLife-Android PR #13. The producer/consumer join is owned by
`docs/architecture/MOBILE-BRIDGE-V1.md`.
