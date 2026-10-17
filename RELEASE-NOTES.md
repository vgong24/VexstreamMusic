# VexStream Music 3.0.0 release notes

## VexLife phone connection

3.0.0 makes taking the VexStream library onto a VexLife Android phone a first-class local product
capability. The source-native 2.2.0 playback/import/Explore behavior remains the preserved base.

### New

- **Connect VexLife phone** surface in the Windows UI.
- Same-Wi-Fi `_vexstream._tcp` discovery with a bounded five-minute first-pairing window.
- Dedicated HTTPS mobile listener; the ordinary desktop admin/UI server stays loopback-only.
- Persistent P-256 TLS certificate identity and six-digit human confirmation code.
- Explicit phone request + desktop **Allow / Deny**.
- Random per-device bearer credentials; desktop persists only the token SHA-256.
- Remembered phones and explicit revoke/forget from either side.
- Sanitized catalog projection with exact media SHA-256 + byte length.
- HTTP Range media streaming for Android Media3.
- Verified Android app-private downloads and restart-safe offline index.
- Certificate-identity route rediscovery when the computer's LAN address changes.
- Explicit **Exit VexStream** control that actually stops the background process.
- Exact build identity so an older same-version process cannot impersonate the current candidate.

### Human path

```text
Windows VexStream
-> click phone icon
-> compare six-digit code

VexLife
-> Find VexStream
-> Codes match · Request access

Windows
-> Allow

VexLife
-> Play / Download
```

ADB reverse remains only a developer/recovery route.

### Security boundary

```text
LOCAL_VEXSTREAM_PAIRING
!= VEXHOME_MEMBERSHIP
!= CANONICAL_HOME_AUTHENTICATION
!= CAPABILITY_LEASE
!= REMOTE_ACCESS_AUTHORITY
```

3.0.0 does not claim away-from-home access authority.

## Preserved 2.2.0 Explore behavior

The multi-axis Explore session remains intact: search-derived factual axes, round-robin projection,
creator cap, one visible provider preview, explicit library promotion, and no normal queue mutation
from metadata/preview.
