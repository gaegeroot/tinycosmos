# Development

Setup notes for working on **Check In** (`checkin`), a Flutter app.

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

The bundle/application ID is `com.gaegeroot.checkin` on every platform. If you ever change it, update all of these places:

- `android/app/build.gradle.kts` (`namespace` and `applicationId`), plus the folder and `package` line of `MainActivity.kt`
- `ios/Runner.xcodeproj/project.pbxproj` and `macos/Runner/Configs/AppInfo.xcconfig`
- `linux/CMakeLists.txt` (`APPLICATION_ID`)

## Docker dev container

The `Dockerfile` builds a container for AI coding agents (Claude Code, OpenCode, ngrok). **It does not include Flutter**, so you can edit code in it but not build or run the app.

For the run command, the security setup and the smoke test, see [setup.md](setup.md).

## Not set up yet

- Version control (the project is not a git repository)
- Release signing for Android and iOS
- CI
- Backend, environment config and secrets
