// Safari theme — premium futuristic (v0.7.3).
//
// Safari's own identity: deep midnight navy, electric cyan +
// turquoise accents, subtle purple highlights, glassmorphism,
// soft neon glows. Dark mode is the star (midnight navy, never
// pure black); light mode is a clean pale-sky companion.
// The Safari planet logo and name stay Safari; the layout, colors
// and shapes are Safari's own — not a WhatsApp clone.
import 'package:flutter/material.dart';

class VyroColors {
  // Brand accents (kept under the old names so existing widgets
  // keep working — they now render in Safari cyan/purple).
  static const Color blue = Color(0xFF22D3EE); // electric cyan
  static const Color purple = Color(0xFF8B5CF6); // violet highlight
  static const Color blueDeep = Color(0xFF0E7490);
  static const Color purpleDeep = Color(0xFF7C3AED);

  static const Color teal = Color(0xFF2DD4BF); // turquoise
  static const Color tealDeep = Color(0xFF0D9488);
  static const Color green = Color(0xFF22D3EE); // cyan (was green)
  static const Color greenDark = Color(0xFF06B6D4);
  static const Color readBlue = Color(0xFF38BDF8);

  /// Cyan -> blue -> purple: the Safari brand gradient.
  static const LinearGradient brandGradient = LinearGradient(
    colors: [Color(0xFF22D3EE), Color(0xFF3B82F6), Color(0xFF8B5CF6)],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );

  static const LinearGradient brandGradientDark = LinearGradient(
    colors: [Color(0xFF06B6D4), Color(0xFF2563EB), Color(0xFF7C3AED)],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );

  /// Story ring: cyan -> purple, used as a border around avatars
  /// with unviewed status updates.
  static const LinearGradient storyRing = LinearGradient(
    colors: [Color(0xFF22D3EE), Color(0xFF8B5CF6)],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );

  /// Soft neon glow used behind hero icons and empty states.
  static const Color glowCyan = Color(0x5522D3EE);
  static const Color glowPurple = Color(0x448B5CF6);
}

/// Safari's premium palette, light and dark.
class WaColors {
  // Light — pale sky.
  static const Color lightHeader = Color(0xFF0B1B33);
  static const Color lightBackground = Color(0xFFF4F8FC);
  static const Color lightChatBackground = Color(0xFFEDF2F7);
  static const Color lightBubbleOut = Color(0xFFCDEFFA);
  static const Color lightBubbleIn = Colors.white;
  static const Color lightText = Color(0xFF0B1B33);
  static const Color lightTextSecondary = Color(0xFF5B6B85);
  static const Color lightDivider = Color(0xFFE2E8F0);
  static const Color lightInputBackground = Colors.white;
  static const Color lightIcon = Color(0xFF5B6B85);
  // Dark — midnight navy (never pure black).
  static const Color darkHeader = Color(0xFF0A1322);
  static const Color darkBackground = Color(0xFF060B18);
  static const Color darkChatBackground = Color(0xFF060B18);
  static const Color darkBubbleOut = Color(0xFF0C3D55);
  static const Color darkBubbleIn = Color(0xFF131F38);
  static const Color darkText = Color(0xFFEAF2FF);
  static const Color darkTextSecondary = Color(0xFF8B9CC4);
  static const Color darkDivider = Color(0xFF1C2A44);
  static const Color darkInputBackground = Color(0xFF111E36);
  static const Color darkIcon = Color(0xFF8B9CC4);
  // Shared.
  static const Color accent = Color(0xFF22D3EE);
  static const Color accentDark = Color(0xFF06B6D4);
  static const Color readBlue = Color(0xFF38BDF8);
  static const Color unreadBadge = Color(0xFF22D3EE);
  static const Color fabLight = Color(0xFF22D3EE);
  static const Color fabDark = Color(0xFF06B6D4);
  // Extra surfaces for glass cards.
  static const Color darkSurface = Color(0xFF0C1526);
  static const Color darkSurface2 = Color(0xFF111E36);
  static const Color darkBorder = Color(0xFF1C2A44);
  static const Color lightSurface = Colors.white;
  static const Color lightBorder = Color(0xFFE2E8F0);
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
      scrolledUnderElevation: 0,
      centerTitle: false,
      titleTextStyle: TextStyle(
        fontSize: 21,
        fontWeight: FontWeight.w800,
        color: Colors.white,
        letterSpacing: 0.4,
      ),
    ),
    navigationBarTheme: NavigationBarThemeData(
      backgroundColor: Colors.white,
      indicatorColor: const Color(0xFFD6F3FB),
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
        borderSide: const BorderSide(color: Color(0xFFE2E8F0)),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(24),
        borderSide: const BorderSide(color: Color(0xFFE2E8F0)),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(24),
        borderSide: const BorderSide(color: Color(0xFF0891B2), width: 1.6),
      ),
    ),
    cardTheme: CardTheme(
      elevation: 0,
      color: WaColors.lightSurface,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(20),
        side: const BorderSide(color: WaColors.lightBorder),
      ),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: const Color(0xFF0891B2),
        foregroundColor: Colors.white,
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(24),
        ),
        textStyle: const TextStyle(fontSize: 15, fontWeight: FontWeight.w700),
      ),
    ),
    floatingActionButtonTheme: const FloatingActionButtonThemeData(
      backgroundColor: WaColors.fabLight,
      foregroundColor: Colors.white,
      elevation: 6,
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
      scrolledUnderElevation: 0,
      centerTitle: false,
      titleTextStyle: TextStyle(
        fontSize: 21,
        fontWeight: FontWeight.w800,
        color: Colors.white,
        letterSpacing: 0.4,
      ),
    ),
    navigationBarTheme: NavigationBarThemeData(
      backgroundColor: WaColors.darkSurface,
      indicatorColor: const Color(0xFF12384A),
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
        borderSide: const BorderSide(color: Color(0xFF1C2A44)),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(24),
        borderSide: const BorderSide(color: Color(0xFF1C2A44)),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(24),
        borderSide: const BorderSide(color: WaColors.accent, width: 1.6),
      ),
    ),
    cardTheme: CardTheme(
      elevation: 0,
      color: WaColors.darkSurface,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(20),
        side: const BorderSide(color: WaColors.darkBorder),
      ),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: WaColors.accentDark,
        foregroundColor: Colors.white,
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(24),
        ),
        textStyle: const TextStyle(fontSize: 15, fontWeight: FontWeight.w700),
      ),
    ),
    floatingActionButtonTheme: const FloatingActionButtonThemeData(
      backgroundColor: WaColors.fabDark,
      foregroundColor: Colors.white,
      elevation: 6,
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
            color: Color(0x4422D3EE),
            blurRadius: 10,
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

/// Gradient story ring around an avatar (Safari cyan -> purple).
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
        boxShadow: [
          BoxShadow(
            color: Color(0x3322D3EE),
            blurRadius: 10,
            offset: Offset(0, 3),
          ),
        ],
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
            color: Color(0x5522D3EE),
            blurRadius: 12,
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

/// Safari avatar: a deterministic cyan/blue/purple gradient (per
/// name) with the first letter — no photo needed. Set [ring] for
/// the cyan-purple status ring around unviewed updates.
class WaAvatar extends StatelessWidget {
  const WaAvatar(
      {super.key, required this.label, this.radius = 26, this.ring = false});

  final String label;
  final double radius;
  final bool ring;

  static const List<List<Color>> _gradients = [
    [Color(0xFF22D3EE), Color(0xFF2563EB)],
    [Color(0xFF8B5CF6), Color(0xFF6366F1)],
    [Color(0xFF2DD4BF), Color(0xFF0891B2)],
    [Color(0xFF38BDF8), Color(0xFF7C3AED)],
    [Color(0xFF06B6D4), Color(0xFF8B5CF6)],
    [Color(0xFF6366F1), Color(0xFF22D3EE)],
    [Color(0xFF0EA5E9), Color(0xFF2DD4BF)],
    [Color(0xFFA78BFA), Color(0xFF22D3EE)],
  ];

  static List<Color> gradientFor(String label) {
    var h = 0;
    for (var i = 0; i < label.length; i++) {
      h = (h * 31 + label.codeUnitAt(i)) & 0x7fffffff;
    }
    return _gradients[h % _gradients.length];
  }

  static Color colorFor(String label) => gradientFor(label).first;

  @override
  Widget build(BuildContext context) {
    final initial = label.isEmpty ? '?' : label.substring(0, 1).toUpperCase();
    final g = gradientFor(label);
    final avatar = Container(
      width: radius * 2,
      height: radius * 2,
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: g,
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        shape: BoxShape.circle,
        boxShadow: const [
          BoxShadow(
            color: Color(0x3322D3EE),
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
    if (!ring) return avatar;
    return Container(
      padding: const EdgeInsets.all(2.5),
      decoration: const BoxDecoration(
        gradient: VyroColors.storyRing,
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

/// A translucent "glass" card: midnight-navy surface, hairline
/// border, soft shadow, generous radius. The building block of
/// Safari's premium surfaces.
class SafariGlassCard extends StatelessWidget {
  const SafariGlassCard({
    super.key,
    required this.child,
    this.padding,
    this.radius = 20,
    this.onTap,
  });

  final Widget child;
  final EdgeInsetsGeometry? padding;
  final double radius;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final card = Container(
      padding: padding ?? const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: dark
            ? WaColors.darkSurface.withOpacity(0.72)
            : Colors.white.withOpacity(0.85),
        borderRadius: BorderRadius.circular(radius),
        border: Border.all(
          color: dark ? WaColors.darkBorder : WaColors.lightBorder,
        ),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(dark ? 0.30 : 0.06),
            blurRadius: 16,
            offset: const Offset(0, 6),
          ),
        ],
      ),
      child: child,
    );
    if (onTap == null) return card;
    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(radius),
        child: card,
      ),
    );
  }
}

/// Premium empty state: a glowing gradient emblem, a short title,
/// a helpful description and a clear action button. Never a bare
/// black void, never fake data — only real actions.
class SafariEmptyState extends StatelessWidget {
  const SafariEmptyState({
    super.key,
    required this.icon,
    required this.title,
    required this.subtitle,
    this.buttonLabel,
    this.onPressed,
    this.emblem,
  });

  final IconData icon;
  final String title;
  final String subtitle;
  final String? buttonLabel;
  final VoidCallback? onPressed;

  /// Optional custom emblem (e.g. the planet logo). When null, a
  /// gradient circle with [icon] is used.
  final Widget? emblem;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final secondary =
        dark ? WaColors.darkTextSecondary : WaColors.lightTextSecondary;
    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.fromLTRB(32, 40, 32, 40),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            emblem ??
                Container(
                  width: 96,
                  height: 96,
                  decoration: const BoxDecoration(
                    gradient: VyroColors.brandGradient,
                    shape: BoxShape.circle,
                    boxShadow: [
                      BoxShadow(
                        color: Color(0x5522D3EE),
                        blurRadius: 28,
                        offset: Offset(0, 8),
                      ),
                      BoxShadow(
                        color: Color(0x338B5CF6),
                        blurRadius: 48,
                        offset: Offset(0, 12),
                      ),
                    ],
                  ),
                  child: Icon(icon, size: 44, color: Colors.white),
                ),
            const SizedBox(height: 24),
            Text(
              title,
              textAlign: TextAlign.center,
              style: TextStyle(
                fontSize: 19,
                fontWeight: FontWeight.w800,
                color: dark ? WaColors.darkText : WaColors.lightText,
                letterSpacing: 0.2,
              ),
            ),
            const SizedBox(height: 8),
            Text(
              subtitle,
              textAlign: TextAlign.center,
              style: TextStyle(fontSize: 14, height: 1.5, color: secondary),
            ),
            if (buttonLabel != null && onPressed != null) ...[
              const SizedBox(height: 24),
              FilledButton.icon(
                onPressed: onPressed,
                icon: const Icon(Icons.add, size: 18),
                label: Text(buttonLabel!),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

/// Safari app bar: midnight navy (dark) / deep navy (light),
/// white title. Pass [titleWidget] for a custom title (e.g. the
/// logo + "Safari" on the main bar, or the chat header with avatar
/// + name + status).
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
