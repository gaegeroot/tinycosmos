# Repository Summary: `checkin`

## Overview
`checkin` ("Check In") is an early-stage Flutter app (version `0.1.0+1`, Dart SDK `^3.13.7`) for groups of friends who answer a weekly prompt and see each other's answers. The only screen so far is a static welcome page in `lib/main.dart`, with "Sign up" and "See how it works" buttons that don't do anything yet. The project is not a git repository.

- **Package name:** `checkin` (`publish_to: none`)
- **Bundle/application ID:** `com.gaegeroot.checkin` (all platforms).
- **Dependencies:** Flutter SDK only; dev deps are `flutter_test` and `flutter_lints ^6.0.0`
- **README:** the default stub ("A new Flutter project.")

## Directory Layout

```
/workspace
├── lib/
│   └── main.dart              # App entry point + static welcome page
├── pubspec.yaml / .lock       # Package manifest and resolved deps
├── analysis_options.yaml      # flutter_lints; excludes build/ and platform dirs
├── Dockerfile                 # Dev container for AI coding agents (see below)
├── README.md                  # Default Flutter stub
├── checkin.iml, .idea/        # IntelliJ/Android Studio project files
├── .dart_tool/                # Generated Dart tooling cache (gitignored)
├── .metadata                  # Flutter project metadata
├── android/                   # Gradle (Kotlin DSL) Android host project
├── ios/                       # Xcode Runner project (Swift, SceneDelegate, SwiftPM ephemeral)
├── macos/                     # Xcode Runner project for macOS desktop
├── linux/                     # CMake + GTK runner (C++)
├── windows/                   # CMake + Win32 runner (C++)
└── web/                       # index.html, manifest.json, icons
```

All six platform folders contain unmodified Flutter template boilerplate, including generated plugin registrants, launcher icons, and launch screens.

## Notable Files
- **`Dockerfile`:** a dev environment based on `node:22-bookworm-slim`. It installs Claude Code, OpenCode, and ngrok, and runs as an unprivileged `agent` user whose UID/GID match the host. It does **not** install the Flutter SDK, so the app can't be built inside this container as-is.
- **`analysis_options.yaml`:** uses `package:flutter_lints/flutter.yaml` and excludes `build/` and every platform directory from analysis.

## Observations / Gaps
- There is no `test/` directory and there are no widget tests.
- The README has no project-specific documentation yet.
