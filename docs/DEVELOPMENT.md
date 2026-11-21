# Development

Setup notes for working on **Tiny Cosmos** (Dart package `tiny_cosmos`), a Flutter app in `apps/mobile/`.

## Prerequisites

| Tool | Version | Needed for |
|---|---|---|
| Flutter SDK | **3.47.9** exactly (pinned in `pubspec.yaml` and `.fvmrc`) | Everything |
| Xcode + CocoaPods | iOS 15.0+ / macOS 12.0+ deployment targets | iOS and macOS builds |
| Android Studio, Android SDK | JDK **17** (Gradle 9.3.1, AGP 9.1.0, Kotlin 2.4.0) | Android builds |
| Chrome | Any recent version | Web builds |

Run `flutter doctor` to see what's still missing on your machine.

## First-time setup

```bash
flutter pub get
```

The Flutter version is pinned, so `flutter pub get` fails on any other version. If you use [FVM](https://fvm.app), run `fvm install` to get the right version, then put `fvm` in front of the commands below (for example, `fvm flutter run`). To upgrade Flutter, change the version in both `pubspec.yaml` and `.fvmrc`.

`android/local.properties` is created by Flutter and holds the path to your local SDK. Don't commit it.

## Running the app

```bash
flutter devices          # list available targets
flutter run              # run on the default device
flutter run -d chrome    # run in the browser
```

The entry point is `lib/main.dart`. IntelliJ / Android Studio users can use the bundled `main.dart` run configuration.

## Checks

```bash
flutter analyze          # lints from package:flutter_lints
dart format .            # format code
flutter test             # no tests exist yet; add them under test/
```

Analysis skips `build/` and all the platform folders (see `analysis_options.yaml`).

## App identifier

The bundle/application ID is `com.gaegeroot.tinycosmos` on every platform, and the desktop executable name is `tiny_cosmos`. If you ever change them, update all of these places (paths relative to `apps/mobile/`):

- **Android:** `android/app/build.gradle.kts` (`namespace` and `applicationId`); the `kotlin/com/gaegeroot/tinycosmos/` folder and the `package` line of `MainActivity.kt` and `TinyCosmosWidgetProvider.kt`; the fully qualified widget receiver name in `android/app/src/main/AndroidManifest.xml`
- **iOS:** `ios/Runner.xcodeproj/project.pbxproj` (app, `.RunnerTests` and `.TinyCosmosWidget` bundle IDs; the widget ID must stay prefixed by the app ID)
- **macOS:** `macos/Runner/Configs/AppInfo.xcconfig` (`PRODUCT_NAME`, `PRODUCT_BUNDLE_IDENTIFIER`); `macos/Runner.xcodeproj/project.pbxproj` (`.RunnerTests` IDs, `tiny_cosmos.app` product and `TEST_HOST`); `macos/Runner.xcodeproj/xcshareddata/xcschemes/Runner.xcscheme` (`BuildableName`)
- **Linux:** `linux/CMakeLists.txt` (`BINARY_NAME`, `APPLICATION_ID`)
- **Windows:** `windows/CMakeLists.txt` (`project`, `BINARY_NAME`); `windows/runner/Runner.rc` (`InternalName`, `OriginalFilename`)

User-facing display names ("Tiny Cosmos") are set separately in the platform manifests, `Info.plist` files, `web/` and the desktop runners.

## Docker dev container

The container setup in `infrastructure/docker/` provides Claude Code, Go, Dart, SQLite, Git, and common build tools for monorepo development. **It intentionally does not include Flutter or mobile SDKs**, so mobile UI builds still run on the host.

For the run command, the security setup and the smoke test, see [setup.md](setup.md).

## Not set up yet

- Version control (the project is not a git repository)
- Release signing for Android and iOS
- CI
- Backend, environment config and secrets
