# Parallel Agent Session Tasks

## Session A

Branch name: spec/initial-scoping
Worktree directory: ../agent-initial-scoping
Task:
Rebrand the existing "checkin" Flutter scaffold to Tiny Cosmos. No new features.
The environment has no Flutter or Dart toolchain, so make edits carefully by hand
and verify by reading and searching, not by running code.
1. Rename the Dart package from `checkin` to `tiny_cosmos` in pubspec.yaml, and
   update every `package:checkin/...` import (including in test/).
2. Replace user-facing "Checkin"/"checkin" text with "Tiny Cosmos": the MaterialApp
   title, welcome screen copy, and platform display names (Android app label,
   iOS CFBundleDisplayName/CFBundleName, web title and manifest name, desktop
   window titles if those platforms exist).
3. Replace the existing styling with the Tiny Cosmos theme in a single theme file
   (e.g. lib/theme/app_theme.dart), defined as named color constants:
     night background #0E1330, surface #161C44, rings/borders #2A3264,
     gold accent #F2C46B, core glow #FFE7A8, secondary text #B9BEDB,
     member colors #F08A6C #5CC8B8 #A99CF0 #6FA8F5.
   Use a dark ColorScheme built from these, using only APIs from the Flutter SDK
   (no new packages). Define heading and body font-family names as constants
   ('Space Grotesk', 'Inter') with a TODO noting the fonts are not yet bundled;
   Flutter falls back to the default font until they are.
4. Update the welcome screen to use the theme: "Tiny Cosmos" title and a short
   tagline, e.g. "A weekly check-in for your people."
5. Replace README.md with the Tiny Cosmos README.
Do NOT change Android applicationId/namespace, the Kotlin/Java package path, or
the iOS bundle identifier. Do NOT add dependencies or edit pubspec.lock.

Files or folders the agent may write to:
- pubspec.yaml (name and description only)
- lib/
- test/
- README.md
- android/app/src/main/AndroidManifest.xml (android:label only)
- ios/Runner/Info.plist (CFBundleDisplayName, CFBundleName only)
- web/index.html, web/manifest.json (title, name, description only)
- macos/, windows/, linux/ window-title and display-name strings only, if present

Files or folders the agent may read but not write to:
- pubspec.lock
- android/app/build.gradle(.kts), android/app/src/main/kotlin (or java)
- ios/Runner.xcodeproj/, ios/Runner/Assets.xcassets/
- android/app/src/main/res/ (launcher icons)
- analysis_options.yaml
- .gitignore

Commands the agent may run:
- rg (or grep) to search the repo, e.g. rg -i "checkin", rg "package:"
- git status, git diff, git add, git commit (no push, no rebase, no force)

Definition of done:
- `rg -i checkin` finds no matches except intentionally out-of-scope platform
  identifiers (applicationId, bundle ID, Kotlin package path) and pubspec.lock.
- pubspec.yaml has `name: tiny_cosmos`; `rg "package:checkin"` returns nothing.
- Every file under lib/ and test/ that imports the theme or renamed package
  has been re-read after editing for correct imports and balanced brackets.
- All colors live in the theme file; `rg "Color\(0x" lib` matches only that file.
- Welcome screen code shows "Tiny Cosmos" and the tagline, using the theme.
- README.md is the Tiny Cosmos README.
- Changes committed on chore/rebrand-tiny-cosmos, and the agent's final message
  lists: files changed, any "checkin" references deliberately left, and anything
  it was unsure would compile.

## Session B

Branch name:
feature/widget-scaffold

Worktree directory:
../agent-widget-scaffold

Task:
Add a very basic, static home screen widget on Android and iOS that shows a
hard-coded text snippet. No data from the Flutter app, no App Groups, no
home_widget package, no updates beyond the static content.
The environment has no Flutter, Dart, Xcode or Gradle toolchain: write code by
hand, keep it minimal, and verify by reading and searching.

Content (both platforms):
- Small label: "This week's question"
- Snippet: "What are you excited about this week?"
- Styling: background #0E1330, label #F2C46B, snippet #F4F5FB.

Android:
1. A Kotlin AppWidgetProvider (e.g. TinyCosmosWidgetProvider) in the app's
   existing Kotlin package directory (do not rename or move the package).
2. A layout in res/layout/ using only RemoteViews-supported views
   (FrameLayout/LinearLayout, TextView).
3. An appwidget-provider definition in res/xml/ for a small 2x2 widget:
   targetCellWidth/targetCellHeight 2, plus minWidth/minHeight 110dp as a
   fallback for older launchers; resizeMode none; updatePeriodMillis 0.
4. A <receiver> entry in AndroidManifest.xml with the APPWIDGET_UPDATE intent
   filter and the provider meta-data.
5. Strings (widget name and description, label, snippet) and colors added to
   res/values/ resources rather than hard-coded in the layout.

iOS:
1. Edit only the Swift files in the existing ios/TinyCosmosWidget/ extension
   that was created in Xcode. Replace the template with a static
   TimelineProvider that returns one entry and the .never reload policy.
2. A SwiftUI view showing the label and snippet in the colors above, using
   .containerBackground(for: .widget) for the background.
3. supportedFamilies limited to [.systemSmall]; configurationDisplayName
   "Tiny Cosmos" and a short description.
4. Remove the template's sample/demo code (emoji entries, intents) if present.

Files or folders the agent may write to:
- android/app/src/main/kotlin/ (or java/): new widget provider file only
- android/app/src/main/res/layout/, res/xml/, res/values/
- android/app/src/main/AndroidManifest.xml (add the receiver only)
- ios/TinyCosmosWidget/ (Swift source files only)

Files or folders the agent may read but not write to:
- Everything else in the repo, including lib/, pubspec.yaml,
  android/app/build.gradle(.kts), ios/Runner/, ios/Runner.xcodeproj/
  (especially project.pbxproj), and ios/TinyCosmosWidget/Info.plist

Commands the agent may run:
- rg (or grep) to search the repo
- git status, git diff, git add, git commit (no push, no rebase, no force)

Definition of done (agent-verifiable):
- All Android files above exist; the manifest receiver's android:name matches
  the provider class's full package and name, and its meta-data resource
  matches the res/xml file name.
- The layout uses only RemoteViews-supported views; all text and colors come
  from resources.
- The iOS widget files declare one Widget with supportedFamilies [.systemSmall],
  a static provider, and no leftover template sample code.
- No files outside the writable list were changed (check with git diff --stat).
- Committed on feature/widget-scaffold. The final message lists files changed
  and anything it was unsure would compile.

Definition of done (you, on device):
- Android: build and run, then long-press the home screen › Widgets ›
  Tiny Cosmos. The small widget can be added and shows the label and snippet.
- iOS: build the Runner scheme, then long-press the home screen › + › Tiny Cosmos.
  The small widget can be added and shows the label and snippet.