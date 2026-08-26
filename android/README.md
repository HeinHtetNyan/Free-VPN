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

`ApiClient.DEV_BASE_URL` points at the real production backend, `https://sy-api.heinh.dev` (updated 2026-08-27 — previously the emulator-only `10.0.2.2` alias).

## Verified on a real physical device (2026-08-27)

Not just "compiles" — installed and used on a real phone over wireless `adb` (`adb pair`/`adb connect`, no cable needed once paired), with real results: real backend connection, real Android VPN permission dialog, real WireGuard handshake through the real LocalToNet relay, real browsing through the tunnel (confirmed via the status bar's VPN key icon and an actual page load). This is the strongest verification this project has had — stronger than `infra/local-test/`, since it exercises code paths (the actual `VpnService`/`GoBackend` integration, real Android permission flow) that a Docker-based test never touches.

This real-device test caught **two real bugs** that neither a compile check nor local-test would have found:
1. `MainActivity.onConnectClick()` called `vpnManager.connect()` inside a nested coroutine `launch{}` not covered by the enclosing `try/catch` — a `BackendException`/timeout from the WireGuard library crashed the whole app instead of surfacing an error. Fixed by keeping the risky call inside the same coroutine body as its `try/catch`, only hopping to `Dispatchers.Main` for UI-only work.
2. The UI set `Connected` immediately after starting the connect flow, before the system permission dialog was even answered.
Both fixed in `MainActivity.kt`.

**Debug builds are now reproducibly signed** — `app/debug.keystore` is committed and pinned explicitly in `build.gradle.kts`'s `signingConfigs`. Before this, each Docker build run generated a fresh random debug keystore, so installing a rebuilt APK over a previous install failed with `INSTALL_FAILED_UPDATE_INCOMPATIBLE` unless you uninstalled first. Standard well-known debug alias/password (`androiddebugkey`/`android`) — not a secret, safe to commit.

## Build status: compiles successfully

This environment has no Android Studio, but it does have Docker — a headless Android SDK build environment (`Dockerfile` in this folder) runs `./gradlew assembleDebug` for real. Along the way (2026-08-26) this caught one real bug reading the code alone wouldn't have: Kotlin 2.0+ needs the separate Compose Compiler Gradle plugin, missing from both `build.gradle.kts` files until that first build run.

To rebuild yourself (from repo root):
```
docker build -t sy-vpn-android-build android/
docker run --rm -v "$(pwd)/android:/project" -w /project --user root sy-vpn-android-build ./gradlew assembleDebug --no-daemon
# fix ownership afterward, since the container writes build/ as root:
docker run --rm -v "$(pwd)/android:/project" alpine sh -c "chown -R $(id -u):$(id -g) /project"
```
The gradle wrapper (`gradlew`, `gradlew.bat`, `gradle/wrapper/gradle-wrapper.jar`) is committed and real — no manual step needed to bootstrap it. Occasionally flaky in a fresh container (`Failed to create Jar file` / a stray `TreeMap$KeySet` error) — a transient Gradle daemon issue in the ephemeral container, not a real problem; just retry.

## Installing on a real device over wireless debugging

1. Phone: Settings → Developer options → enable **Wireless debugging**.
2. `adb mdns services` / `adb devices -l` auto-discovers a phone already paired once, on the same network — no cable needed on later runs. First-time pairing needs `adb pair <ip>:<port> <code>` with the code shown on the phone.
3. Install: `adb install -r android/app/build/outputs/apk/debug/app-debug.apk` (add `adb uninstall com.syvpn.app` first only if switching to/from an APK signed with a different keystore than the one currently installed).

## Known gaps / TODOs left in the code

- `ads/AdsterraBannerAd.kt` — ad tag markup is a placeholder shape, not a real Adsterra snippet. Needs a real zone ID + the actual tag from Adsterra's dashboard (`../docs/OPEN_QUESTIONS.md`).
- The launcher icon (`res/drawable/ic_launcher_*.xml`) is a placeholder shield mark, not a final logo — fine as-is for now (matches the SY VPN name), revisit if you want real branding/logo design later.
