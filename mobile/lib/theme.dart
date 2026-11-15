// Safari theme — WhatsApp-style (v0.7).
//
// WhatsApp look: teal-green header (#008069), bright green accent
// (#25D366), white / dark navy surfaces, plain chat background
// (#ECE5DD light, #0B141A dark), green outgoing bubbles (#D9FDD3
// light, #005C4B dark). The Safari swift logo and name stay Safari;
// the layout, colors and shapes follow WhatsApp.
import 'package:flutter/material.dart';

class VyroColors {
  // WhatsApp greens (kept under the old names so existing widgets
  // keep working — they now render in WhatsApp colors).
  static const Color blue = Color(0xFF008069); // header teal
  static const Color purple = Color(0xFF25D366); // accent green
  static const Color blueDeep = Color(0xFF075E54);
  static const Color purpleDeep = Color(0xFF00A884);

  static const Color teal = Color(0xFF008069);
  static const Color tealDeep = Color(0xFF075E54);
  static const Color green = Color(0xFF25D366);
  static const Color greenDark = Color(0xFF00A884);
  static const Color readBlue = Color(0xFF53BDEB);

  static const LinearGradient brandGradient = LinearGradient(
    colors: [teal, green],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );

  static const LinearGradient brandGradientDark = LinearGradient(
    colors: [tealDeep, greenDark],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );

  /// Story ring: WhatsApp green, used as a border around avatars
  /// with unviewed status updates.
  static const LinearGradient storyRing = LinearGradient(
    colors: [green, greenDark],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );
}

/// Full WhatsApp palette, light and dark.
class WaColors {
  // Light.
  static const Color lightHeader = Color(0xFF008069);
  static const Color lightBackground = Colors.white;
  static const Color lightChatBackground = Color(0xFFECE5DD);
  static const Color lightBubbleOut = Color(0xFFD9FDD3);
  static const Color lightBubbleIn = Colors.white;
  static const Color lightText = Color(0xFF111B21);
  static const Color lightTextSecondary = Color(0xFF667781);
  static const Color lightDivider = Color(0xFFE9EDEF);
  static const Color lightInputBackground = Colors.white;
  static const Color lightIcon = Color(0xFF54656F);
  // Dark.
  static const Color darkHeader = Color(0xFF1F2C34);
  static const Color darkBackground = Color(0xFF0B141A);
  static const Color darkChatBackground = Color(0xFF0B141A);
  static const Color darkBubbleOut = Color(0xFF005C4B);
  static const Color darkBubbleIn = Color(0xFF202C33);
  static const Color darkText = Color(0xFFE9EDEF);
  static const Color darkTextSecondary = Color(0xFF8696A0);
  static const Color darkDivider = Color(0xFF222D34);
  static const Color darkInputBackground = Color(0xFF1F2C34);
  static const Color darkIcon = Color(0xFF8696A0);
  // Shared.
  static const Color accent = Color(0xFF25D366);
  static const Color accentDark = Color(0xFF00A884);
  static const Color readBlue = Color(0xFF53BDEB);
  static const Color unreadBadge = Color(0xFF25D366);
  static const Color fabLight = Color(0xFF25D366);
  static const Color fabDark = Color(0xFF00A884);
}

ThemeData vyroLightTheme() {
  final scheme = ColorScheme.fromSeed(
    seedColor: WaColors.lightHeader,
    brightness: Brightness.light,
  );
  return ThemeData(
    colorScheme: scheme,
    useMaterial3: true,
    scaffoldBackgroundColor: WaColors.lightBackground,
    fontFamily: 'Roboto',
    appBarTheme: const AppBarTheme(
      backgroundColor: WaColors.lightHeader,
      foregroundColor: Colors.white,
      elevation: 0,
      centerTitle: false,
      titleTextStyle: TextStyle(
        fontSize: 20,
        fontWeight: FontWeight.w700,
        color: Colors.white,
        letterSpacing: 0.2,
      ),
    ),
    navigationBarTheme: NavigationBarThemeData(
      backgroundColor: Colors.white,
      indicatorColor: const Color(0xFFE7F8E9),
      labelTextStyle: WidgetStateProperty.all(
        const TextStyle(fontSize: 12, fontWeight: FontWeight.w600),
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: Colors.white,
      contentPadding: const EdgeInsets.symmetric(horizontal: 18, vertical: 14),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(24),
        borderSide: BorderSide(color: Colors.grey.shade300),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(24),
        borderSide: BorderSide(color: Colors.grey.shade300),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(24),
        borderSide: const BorderSide(color: WaColors.lightHeader, width: 1.6),
      ),
    ),
    cardTheme: CardTheme(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
      ),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: WaColors.accent,
        foregroundColor: Colors.white,
        padding: const EdgeInsets.symmetric(horizontal: 22, vertical: 14),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(24),
        ),
        textStyle: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
      ),
    ),
    floatingActionButtonTheme: const FloatingActionButtonThemeData(
      backgroundColor: WaColors.fabLight,
      foregroundColor: Colors.white,
      elevation: 4,
    ),
    dividerTheme: const DividerThemeData(color: WaColors.lightDivider),
    listTileTheme: const ListTileThemeData(
      contentPadding: EdgeInsets.symmetric(horizontal: 16, vertical: 4),
    ),
  );
}

ThemeData vyroDarkTheme() {
  final scheme = ColorScheme.fromSeed(
    seedColor: WaColors.darkHeader,
    brightness: Brightness.dark,
  );
  return ThemeData(
    colorScheme: scheme,
    useMaterial3: true,
    scaffoldBackgroundColor: WaColors.darkBackground,
    fontFamily: 'Roboto',
    appBarTheme: const AppBarTheme(
      backgroundColor: WaColors.darkHeader,
      foregroundColor: Colors.white,
      elevation: 0,
      centerTitle: false,
      titleTextStyle: TextStyle(
        fontSize: 20,
        fontWeight: FontWeight.w700,
        color: Colors.white,
        letterSpacing: 0.2,
      ),
    ),
    navigationBarTheme: NavigationBarThemeData(
      backgroundColor: WaColors.darkHeader,
      indicatorColor: const Color(0xFF1A3A34),
      labelTextStyle: WidgetStateProperty.all(
        const TextStyle(fontSize: 12, fontWeight: FontWeight.w600),
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: WaColors.darkInputBackground,
      contentPadding: const EdgeInsets.symmetric(horizontal: 18, vertical: 14),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(24),
        borderSide: const BorderSide(color: Color(0xFF2E323C)),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(24),
        borderSide: const BorderSide(color: Color(0xFF2E323C)),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(24),
        borderSide: const BorderSide(color: WaColors.accentDark, width: 1.6),
      ),
    ),
    cardTheme: CardTheme(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
      ),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: WaColors.accentDark,
        foregroundColor: Colors.white,
        padding: const EdgeInsets.symmetric(horizontal: 22, vertical: 14),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(24),
        ),
        textStyle: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
      ),
    ),
    floatingActionButtonTheme: const FloatingActionButtonThemeData(
      backgroundColor: WaColors.fabDark,
      foregroundColor: Colors.white,
      elevation: 4,
    ),
    dividerTheme: const DividerThemeData(color: WaColors.darkDivider),
    listTileTheme: const ListTileThemeData(
      contentPadding: EdgeInsets.symmetric(horizontal: 16, vertical: 4),
    ),
  );
}

/// Gradient avatar circle with an initial letter.
class GradientAvatar extends StatelessWidget {
  const GradientAvatar(
      {super.key, required this.label, this.radius = 26, this.gradient});

  final String label;
  final double radius;
  final LinearGradient? gradient;

  @override
  Widget build(BuildContext context) {
    final initial = label.isEmpty ? '?' : label.substring(0, 1).toUpperCase();
    return Container(
      width: radius * 2,
      height: radius * 2,
      decoration: BoxDecoration(
        gradient: gradient ?? VyroColors.brandGradient,
        shape: BoxShape.circle,
        boxShadow: const [
          BoxShadow(
            color: Color(0x332563EB),
            blurRadius: 8,
            offset: Offset(0, 3),
          ),
        ],
      ),
      alignment: Alignment.center,
      child: Text(
        initial,
        style: TextStyle(
          color: Colors.white,
          fontWeight: FontWeight.bold,
          fontSize: radius * 0.75,
        ),
      ),
    );
  }
}

/// Gradient story ring around an avatar (Telegram-style, VYRO colors).
class StoryRing extends StatelessWidget {
  const StoryRing({super.key, required this.child, this.ringWidth = 3});

  final Widget child;
  final double ringWidth;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: EdgeInsets.all(ringWidth),
      decoration: const BoxDecoration(
        gradient: VyroColors.storyRing,
        shape: BoxShape.circle,
      ),
      child: Container(
        padding: const EdgeInsets.all(2.5),
        decoration: BoxDecoration(
          color: Theme.of(context).scaffoldBackgroundColor,
          shape: BoxShape.circle,
        ),
        child: child,
      ),
    );
  }
}

/// Circular send button with the brand gradient.
class GradientIconButton extends StatelessWidget {
  const GradientIconButton(
      {super.key, required this.icon, required this.onPressed, this.size = 52});

  final IconData icon;
  final VoidCallback onPressed;
  final double size;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: size,
      height: size,
      decoration: const BoxDecoration(
        gradient: VyroColors.brandGradient,
        shape: BoxShape.circle,
        boxShadow: [
          BoxShadow(
            color: Color(0x447C3AED),
            blurRadius: 10,
            offset: Offset(0, 4),
          ),
        ],
      ),
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          customBorder: const CircleBorder(),
          onTap: onPressed,
          child: Icon(icon, color: Colors.white, size: 26),
        ),
      ),
    );
  }
}

/// WhatsApp-style avatar: a solid muted color (deterministic per
/// name) with the first letter — no photo needed. Set [ring] for the
/// green status ring around unviewed updates.
class WaAvatar extends StatelessWidget {
  const WaAvatar(
      {super.key, required this.label, this.radius = 26, this.ring = false});

  final String label;
  final double radius;
  final bool ring;

  static const List<Color> _colors = [
    Color(0xFF008069),
    Color(0xFF00A884),
    Color(0xFF54656F),
    Color(0xFF075E54),
    Color(0xFF128C7E),
    Color(0xFF25D366),
    Color(0xFF8696A0),
    Color(0xFF53BDEB),
  ];

  static Color colorFor(String label) {
    var h = 0;
    for (var i = 0; i < label.length; i++) {
      h = (h * 31 + label.codeUnitAt(i)) & 0x7fffffff;
    }
    return _colors[h % _colors.length];
  }

  @override
  Widget build(BuildContext context) {
    final initial = label.isEmpty ? '?' : label.substring(0, 1).toUpperCase();
    final avatar = CircleAvatar(
      radius: radius,
      backgroundColor: colorFor(label),
      child: Text(
        initial,
        style: TextStyle(
          color: Colors.white,
          fontWeight: FontWeight.bold,
          fontSize: radius * 0.75,
        ),
      ),
    );
    if (!ring) return avatar;
    return Container(
      padding: const EdgeInsets.all(2.5),
      decoration: const BoxDecoration(
        color: WaColors.accent,
        shape: BoxShape.circle,
      ),
      child: Container(
        padding: const EdgeInsets.all(2),
        decoration: BoxDecoration(
          color: Theme.of(context).scaffoldBackgroundColor,
          shape: BoxShape.circle,
        ),
        child: avatar,
      ),
    );
  }
}

/// WhatsApp-style app bar: solid teal (light) / dark header (dark),
/// white title. Pass [titleWidget] for a custom title (e.g. the chat
/// header with avatar + name + status).
PreferredSizeWidget vyroAppBar(
  BuildContext context, {
  String? title,
  Widget? titleWidget,
  required List<Widget> actions,
  Widget? leading,
  PreferredSizeWidget? bottom,
}) {
  return AppBar(
    leading: leading,
    title: titleWidget ?? (title == null ? null : Text(title)),
    actions: actions,
    bottom: bottom,
  );
}

