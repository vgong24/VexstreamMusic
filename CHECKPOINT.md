# VexStream Music 3.0.0 checkpoint

```text
continuity=[VXG RealForever]
entryClass=SAME_LANE_CAPABILITY_SUCCESSOR
product=VexStream Music
preservedBase=2.2.0 Explore multi-axis refinement
currentCandidate=3.0.0
scope=VEXLIFE_LOCAL_MOBILE_STREAM_OFFLINE
desktopAdminListener=LOOPBACK_ONLY
mobileListener=DEDICATED_HTTPS
mobileDiscovery=MDNS_VEXSTREAM_TCP
pairing=HUMAN_CODE_COMPARE_PLUS_DESKTOP_APPROVAL
credentialPersistence=DESKTOP_HASH_ONLY__ANDROID_KEYSTORE_ENCRYPTED
mobileCatalog=SANITIZED
mediaIdentity=SHA256_PLUS_LENGTH
streaming=HTTP_RANGE_TO_ANDROID_MEDIA3
offline=VERIFIED_APP_PRIVATE_PLUS_DURABLE_INDEX
rememberedIdentity=TLS_CERTIFICATE_SHA256
routeRediscovery=SUPPORTED
developerFallback=ADB_REVERSE_PRESERVED
backgroundExit=EXPLICIT_UI_EXIT
sameVersionReuse=VERSION_PLUS_BUILD_REF
remoteHomeAuthority=NOT_CLAIMED
```

Observed before 3.0 formation: the development ADB bridge streamed the real desktop library to a
physical Samsung phone and played audio through the phone.

Current machine proof: Windows source/build/mobile regressions and Android APK/source/unit proof are
green on their current branch heads.

Remaining lived 3.0 acceptance edge:

```text
same-Wi-Fi pair without ADB
-> stream
-> download one real track
-> stop desktop
-> local-only playback
-> restart VexLife
-> offline copy restored
```
