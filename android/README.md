# android

Native Android app (Kotlin + Jetpack Compose). See [`../docs/MOBILE.md`](../docs/MOBILE.md) for the approach and UX priorities.

## Current state (implemented 2026-08-26)

A real Gradle project. App name **SY VPN**, package `com.syvpn.app` (see `../docs/DECISIONS.md` 2026-08-26 for the naming rationale):

```
android/
├── settings.gradle.kts / build.gradle.kts / gradle.properties
├── gradlew / gradlew.bat / gradle/wrapper/   real, generated wrapper — no bootstrap step needed
├── Dockerfile                                headless Android SDK build image, see "Build status" below
└── app/
    ├── build.gradle.kts        Compose (+ Compose Compiler plugin), WireGuard tunnel lib, minimal networking deps
    ├── src/main/AndroidManifest.xml
    ├── src/main/res/           placeholder adaptive launcher icon (shield+checkmark), strings, colors
    └── src/main/java/com/syvpn/app/
        ├── MainActivity.kt      wires everything together
        ├── ui/ConnectScreen.kt  the whole core flow: location picker, connect/disconnect, status
        ├── ui/theme/Theme.kt
        ├── data/ApiClient.kt        talks to backend/ (register, locations, connect)
        ├── data/DeviceIdentity.kt   local anonymous device ID + token storage
        ├── vpn/VpnConnectionManager.kt  wraps WireGuard's official GoBackend
        └── ads/AdsterraBannerAd.kt      WebView ad, Banner-only per Play Store compliance
```

`ApiClient.DEV_BASE_URL` points at `http://10.0.2.2:8090` (the Android emulator's alias for your dev machine's localhost) — matches running `backend/` locally with `PORT=8090 go run ./cmd/server`. Replace with the real control-plane URL once a central server is deployed.

## Build status: compiles successfully (verified 2026-08-26)

This environment has no Android Studio, but it does have Docker — so a headless Android SDK build environment (`Dockerfile` in this folder) was used to actually run `./gradlew assembleDebug` and confirm the project builds. It does: **`BUILD SUCCESSFUL`, produced a real 20MB `app-debug.apk`.** Along the way this caught and fixed one real bug that couldn't have been found by reading the code: Kotlin 2.0+ requires the separate Compose Compiler Gradle plugin, which was missing from both `build.gradle.kts` files until this build run.

This confirms the code compiles and the WireGuard native libraries (`libwg.so` etc.) link correctly — it does **not** confirm the UI behaves correctly or that the app runs (that needs an emulator/device with a display; see below).

To rebuild yourself (from repo root):
```
docker build -t sy-vpn-android-build android/
docker run --rm -v "$(pwd)/android:/project" -w /project --user root sy-vpn-android-build ./gradlew assembleDebug --no-daemon
# fix ownership afterward, since the container writes build/ as root:
docker run --rm -v "$(pwd)/android:/project" alpine sh -c "chown -R $(id -u):$(id -g) /project"
```
The gradle wrapper (`gradlew`, `gradlew.bat`, `gradle/wrapper/gradle-wrapper.jar`) is committed and real — no manual step needed to bootstrap it.

## Opening this project in Android Studio (needed to actually run/see the UI)

1. Open `android/` in Android Studio — the wrapper is already in place, so it should sync immediately.
2. Run `backend/` locally first (`cd ../backend && PORT=8090 go run ./cmd/server`) so the emulator has something to talk to.
3. Run on an emulator or device. This machine has KVM available (`/dev/kvm`), so an emulator would have hardware acceleration if Android Studio is later installed here — not attempted yet, since headless UI testing needs more setup (virtual display, adb) than the compile check above.

## Known gaps / TODOs left in the code

- `ads/AdsterraBannerAd.kt` — ad tag markup is a placeholder shape, not a real Adsterra snippet. Needs a real zone ID + the actual tag from Adsterra's dashboard (`../docs/OPEN_QUESTIONS.md`).
- `data/ApiClient.kt` `DEV_BASE_URL` — needs to become a real deployed URL once a central server exists.
- The WireGuard config the backend currently returns has a placeholder `Endpoint`/server public key until `../infra/scripts/setup-central-server.sh` has been run for real (see `../docs/BACKEND.md`) — the mechanism itself (key generation, live peer registration, the tunnel, NAT relay) is proven working end-to-end via `../infra/local-test/`, this is purely "no real server deployed yet," not an unverified code path.
- The launcher icon (`res/drawable/ic_launcher_*.xml`) is a placeholder shield mark, not a final logo — fine as-is for now (matches the SY VPN name), revisit if you want real branding/logo design later.
