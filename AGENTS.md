# VexStream Music — fresh-entry router

`[VXG RealForever]`

This file routes engineering work without replacing the product-facing root README.

## Enter

1. Read [README.md](README.md) for the current product, installation and lived-user contract.
2. For V4 / V2 architecture work, read [v2/README.md](v2/README.md).
3. Then read [v2/architecture/README.md](v2/architecture/README.md) before changing feature ownership
   or translating a pattern from another repository.
4. Descend only into the exact feature/domain and proof implicated by the task.

## Architecture lesson

VexStream intentionally learns from VexLife-Android without becoming an Android/Kotlin copy.

The local architecture entry teaches the reusable lesson:

~~~text
purpose
-> domain meaning
-> first-principle relationship
-> VexStream-native form
-> browser/desktop manifestation
-> proof
~~~

Examples:

~~~text
Playback
    traversal dynamics through the current media timeline

Queue
    traversal ordering

Library
    current truth + reconciliation / continuity

Connection
    manifestation + provider observation
~~~

Permanent:

~~~text
SAME FIRST-PRINCIPLE RELATIONSHIP
!= SAME DOMAIN
!= SAME OWNER
!= SAME IMPLEMENTATION

INTENTION BLUEPRINT
!= FRAMEWORK DEPENDENCY
~~~

Use that lesson to understand unfamiliar source before deciding source shape. Do not create generic
Terrain/Traversal/etc. abstractions merely because the primitive meaning is reusable in explanation.

## Existing-product boundary

V1 / V3 lived behavior remains evidence and product oracle where current V2/V4 source says so.
Architecture work must not silently redefine the lived product merely because a cleaner local form is
available.

## Proof

Test the transformation at its rightful owner. Browser, provider, persistence and lived-product proof
are different claims; follow the exact proof surface owned by the changed feature.

<!-- [VXG RealForever][VEXSTREAM][FRESH-ENTRY] -->
