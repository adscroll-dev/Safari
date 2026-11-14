# flutter_webrtc — VYRO fork (Android only)

Upstream: `flutter_webrtc` 0.12.12+hotfix.1
(https://github.com/cloudwebrtc/flutter-webrtc), copied from the pub
cache. Only the Android native code (`android/`), the Dart API
(`lib/`), and `assets/` are vendored; the iOS/macOS/Windows/Linux
native code and the 97 MB `third_party/` prebuilt binaries were left
out because VYRO's mobile build target right now is the Android APK
(iOS needs a Mac with Xcode anyway).

## Why a fork exists

Flutter 3.24.5's Android embedding changed
`TextureRegistry.SurfaceProducer.Callback` to

```java
interface Callback {
  void onSurfaceCreated();
  void onSurfaceDestroyed();
}
```

No released flutter_webrtc version implements that shape: 0.12.x /
0.13.0 implement `{onSurfaceAvailable, onSurfaceDestroyed}`, and
0.13.1+ / 1.x implement `{onSurfaceAvailable, onSurfaceCleanup}` (for
a newer engine). So the stock plugin does not compile against Flutter
3.24.5:

```
SurfaceTextureRenderer.java:118: error: <anonymous ...> is not abstract
  and does not override abstract method onSurfaceCreated() in Callback
SurfaceTextureRenderer.java:119: error: method does not override or
  implement a method from a supertype
```

## What changed (2 lines)

`android/src/main/java/com/cloudwebrtc/webrtc/SurfaceTextureRenderer.java`,
in `surfaceCreated()`:

- the anonymous `SurfaceProducer.Callback` now implements
  `onSurfaceCreated()` (empty body — the old `onSurfaceAvailable()`
  body was already a no-op comment) instead of `onSurfaceAvailable()`;
- `onSurfaceDestroyed()` is unchanged.

## How to un-fork

When VYRO upgrades to a Flutter whose engine uses the
`{onSurfaceAvailable, onSurfaceCleanup}` Callback (Flutter 3.27+),
switch back to the stock package: delete this directory, remove the
`dependency_overrides` entry for `flutter_webrtc` in
`mobile/pubspec.yaml`, and set `flutter_webrtc` to a 1.x version.
