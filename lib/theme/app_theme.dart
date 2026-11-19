import 'package:flutter/material.dart';

/// Tiny Cosmos color palette.
///
/// Every color used by the app is defined here. Widgets should read colors
/// from [Theme.of] (the [ColorScheme]) or from these constants, never from
/// inline `Color(...)` literals.
abstract final class AppColors {
  /// Night background behind every screen.
  static const Color night = Color(0xFF0E1330);

  /// Raised surfaces such as cards and sheets.
  static const Color surface = Color(0xFF161C44);

  /// Orbit rings, dividers and borders.
  static const Color ring = Color(0xFF2A3264);

  /// Warm gold accent for primary actions.
  static const Color gold = Color(0xFFF2C46B);

  /// Glow of the shared center of the cosmos.
  static const Color coreGlow = Color(0xFFFFE7A8);

  /// Secondary text.
  static const Color secondaryText = Color(0xFFB9BEDB);

  /// One color per group member, assigned in order.
  static const Color memberCoral = Color(0xFFF08A6C);
  static const Color memberTeal = Color(0xFF5CC8B8);
  static const Color memberLavender = Color(0xFFA99CF0);
  static const Color memberBlue = Color(0xFF6FA8F5);

  static const List<Color> members = [
    memberCoral,
    memberTeal,
    memberLavender,
    memberBlue,
  ];
}

/// Tiny Cosmos font families.
///
/// TODO: Bundle Space Grotesk and Inter as assets and declare them under
/// `flutter: fonts:` in pubspec.yaml. Until then Flutter falls back to the
/// platform default font.
abstract final class AppFonts {
  static const String heading = 'Space Grotesk';
  static const String body = 'Inter';
}

/// The Tiny Cosmos app theme.
abstract final class AppTheme {
  static const ColorScheme colorScheme = ColorScheme.dark(
    primary: AppColors.gold,
    onPrimary: AppColors.night,
    secondary: AppColors.coreGlow,
    onSecondary: AppColors.night,
    surface: AppColors.surface,
    onSurface: Colors.white,
    onSurfaceVariant: AppColors.secondaryText,
    outline: AppColors.ring,
    outlineVariant: AppColors.ring,
  );

  static ThemeData get dark => ThemeData(
        useMaterial3: true,
        brightness: Brightness.dark,
        colorScheme: colorScheme,
        scaffoldBackgroundColor: AppColors.night,
        dividerColor: AppColors.ring,
        fontFamily: AppFonts.body,
        textTheme: const TextTheme(
          displayLarge: TextStyle(fontFamily: AppFonts.heading),
          displayMedium: TextStyle(fontFamily: AppFonts.heading),
          displaySmall: TextStyle(fontFamily: AppFonts.heading),
          headlineLarge: TextStyle(fontFamily: AppFonts.heading),
          headlineMedium: TextStyle(fontFamily: AppFonts.heading),
          headlineSmall: TextStyle(fontFamily: AppFonts.heading),
          titleLarge: TextStyle(fontFamily: AppFonts.heading),
        ),
      );
}
