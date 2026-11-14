// VYRO premium theme (v0.6).
//
// Blue-to-purple brand gradient (#2563EB -> #7C3AED), clean
// typography, generous spacing, rounded everything. Layout stays
// familiar (Telegram/WhatsApp-like chat list + bubbles) but visibly
// more polished. Light and dark themes are both first-class.
import 'dart:math' as math;
import 'dart:ui';

import 'package:flutter/material.dart';

class VyroColors {
  static const Color blue = Color(0xFF2563EB);
  static const Color purple = Color(0xFF7C3AED);
  static const Color blueDeep = Color(0xFF1E40AF);
  static const Color purpleDeep = Color(0xFF6D28D9);

  static const LinearGradient brandGradient = LinearGradient(
    colors: [blue, purple],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );

  static const LinearGradient brandGradientDark = LinearGradient(
    colors: [blueDeep, purpleDeep],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );

  /// Story ring: the same gradient, used as a border around avatars.
  static const LinearGradient storyRing = LinearGradient(
    colors: [Color(0xFF38BDF8), blue, purple, Color(0xFFC084FC)],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );
}

ThemeData vyroLightTheme() {
  final scheme = ColorScheme.fromSeed(
    seedColor: VyroColors.blue,
    brightness: Brightness.light,
  );
  return ThemeData(
    colorScheme: scheme,
    useMaterial3: true,
    scaffoldBackgroundColor: const Color(0xFFF6F7FB),
    fontFamily: 'Roboto',
    appBarTheme: const AppBarTheme(
      backgroundColor: Colors.transparent,
      foregroundColor: Colors.white,
      elevation: 0,
      centerTitle: false,
      titleTextStyle: TextStyle(
        fontSize: 21,
        fontWeight: FontWeight.w700,
        color: Colors.white,
        letterSpacing: 0.2,
      ),
    ),
    navigationBarTheme: NavigationBarThemeData(
      backgroundColor: Colors.white,
      indicatorColor: VyroColors.blue.withOpacity(0.14),
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
        borderSide: const BorderSide(color: VyroColors.blue, width: 1.6),
      ),
    ),
    cardTheme: CardTheme(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(20),
      ),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: VyroColors.blue,
        foregroundColor: Colors.white,
        padding: const EdgeInsets.symmetric(horizontal: 22, vertical: 14),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
        ),
        textStyle: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
      ),
    ),
    floatingActionButtonTheme: const FloatingActionButtonThemeData(
      backgroundColor: VyroColors.purple,
      foregroundColor: Colors.white,
      elevation: 4,
    ),
    dividerTheme: DividerThemeData(color: Colors.grey.shade200),
    listTileTheme: const ListTileThemeData(
      contentPadding: EdgeInsets.symmetric(horizontal: 16, vertical: 4),
    ),
  );
}

ThemeData vyroDarkTheme() {
  final scheme = ColorScheme.fromSeed(
    seedColor: VyroColors.purple,
    brightness: Brightness.dark,
  );
  return ThemeData(
    colorScheme: scheme,
    useMaterial3: true,
    scaffoldBackgroundColor: const Color(0xFF0F1115),
    fontFamily: 'Roboto',
    appBarTheme: const AppBarTheme(
      backgroundColor: Colors.transparent,
      foregroundColor: Colors.white,
      elevation: 0,
      centerTitle: false,
      titleTextStyle: TextStyle(
        fontSize: 21,
        fontWeight: FontWeight.w700,
        color: Colors.white,
        letterSpacing: 0.2,
      ),
    ),
    navigationBarTheme: NavigationBarThemeData(
      backgroundColor: const Color(0xFF17191F),
      indicatorColor: VyroColors.purple.withOpacity(0.25),
      labelTextStyle: WidgetStateProperty.all(
        const TextStyle(fontSize: 12, fontWeight: FontWeight.w600),
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: const Color(0xFF1E2128),
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
        borderSide: const BorderSide(color: VyroColors.purple, width: 1.6),
      ),
    ),
    cardTheme: CardTheme(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(20),
      ),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: VyroColors.purple,
        foregroundColor: Colors.white,
        padding: const EdgeInsets.symmetric(horizontal: 22, vertical: 14),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
        ),
        textStyle: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
      ),
    ),
    floatingActionButtonTheme: const FloatingActionButtonThemeData(
      backgroundColor: VyroColors.purple,
      foregroundColor: Colors.white,
      elevation: 4,
    ),
    dividerTheme: const DividerThemeData(color: Color(0xFF23262E)),
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

/// AppBar with the brand gradient (light or dark variant).
PreferredSizeWidget vyroAppBar(
  BuildContext context, {
  required String title,
  required List<Widget> actions,
  Widget? leading,
}) {
  final dark = Theme.of(context).brightness == Brightness.dark;
  return AppBar(
    leading: leading,
    title: Text(title),
    actions: actions,
    flexibleSpace: Container(
      decoration: BoxDecoration(
        gradient:
            dark ? VyroColors.brandGradientDark : VyroColors.brandGradient,
      ),
    ),
  );
}

/// --- VYRO "virtual world" layer -------------------------------------
/// The world feel: a slowly drifting animated gradient sky, glassy
/// translucent cards, and one "atmosphere" (gradient pair) per chat /
/// space, derived deterministically from its id — so every group or
/// channel feels like its own little world, and the same chat always
/// has the same atmosphere. All of it is decoration over the real
/// chat list: no fake chats, no fake buttons.

/// One atmosphere: a pair of colors for a space's gradient.
class Atmosphere {
  const Atmosphere(this.a, this.b, this.label);
  final Color a;
  final Color b;
  final String label;
}

const List<Atmosphere> _atmospheres = [
  Atmosphere(Color(0xFF0EA5E9), Color(0xFF7C3AED), 'Ocean Nebula'),
  Atmosphere(Color(0xFFF59E0B), Color(0xFFEC4899), 'Sunset Dunes'),
  Atmosphere(Color(0xFF10B981), Color(0xFF0EA5E9), 'Emerald Lagoon'),
  Atmosphere(Color(0xFF8B5CF6), Color(0xFFEC4899), 'Violet Bloom'),
  Atmosphere(Color(0xFF6366F1), Color(0xFF22D3EE), 'Aurora Sky'),
  Atmosphere(Color(0xFFF43F5E), Color(0xFFF59E0B), 'Ember Valley'),
];

/// Deterministic atmosphere for a chat/space id.
Atmosphere atmosphereFor(String id) {
  var h = 0;
  for (var i = 0; i < id.length; i++) {
    h = (h * 31 + id.codeUnitAt(i)) & 0x7fffffff;
  }
  return _atmospheres[h % _atmospheres.length];
}

LinearGradient atmosphereGradient(String id) {
  final at = atmosphereFor(id);
  return LinearGradient(
    colors: [at.a, at.b],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );
}

/// Animated world sky: a deep gradient with three slowly drifting
/// glowing orbs. Pure decoration, driven by one repeating controller.
class WorldBackground extends StatefulWidget {
  const WorldBackground({super.key, this.child});

  final Widget? child;

  @override
  State<WorldBackground> createState() => _WorldBackgroundState();
}

class _WorldBackgroundState extends State<WorldBackground>
    with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(
      vsync: this,
      duration: const Duration(seconds: 24),
    )..repeat();
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    return AnimatedBuilder(
      animation: _ctrl,
      builder: (context, _) {
        final t = _ctrl.value * 2 * math.pi;
        return Container(
          decoration: BoxDecoration(
            gradient: LinearGradient(
              colors: dark
                  ? const [Color(0xFF14102A), Color(0xFF0F1115)]
                  : const [Color(0xFFE8EFFF), Color(0xFFF6F7FB)],
              begin: Alignment.topCenter,
              end: Alignment.bottomCenter,
            ),
          ),
          child: Stack(
            children: [
              _orb(0.15 + 0.06 * math.sin(t), 0.10, 220,
                  VyroColors.blue.withOpacity(dark ? 0.35 : 0.25)),
              _orb(0.75 + 0.05 * math.cos(t * 0.8), 0.30, 180,
                  VyroColors.purple.withOpacity(dark ? 0.35 : 0.22)),
              _orb(0.45 + 0.07 * math.sin(t * 0.6 + 2), 0.75, 200,
                  const Color(0xFF38BDF8).withOpacity(dark ? 0.28 : 0.18)),
              if (widget.child != null) widget.child!,
            ],
          ),
        );
      },
    );
  }

  Widget _orb(double fx, double fy, double size, Color color) {
    return Positioned(
      left: fx * 400 - size / 2,
      top: fy * 300,
      child: Container(
        width: size,
        height: size,
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          gradient: RadialGradient(
            colors: [color, color.withOpacity(0)],
          ),
        ),
      ),
    );
  }
}

/// Glassy translucent card (frosted glass) for world surfaces.
class GlassCard extends StatelessWidget {
  const GlassCard({super.key, required this.child, this.padding});

  final Widget child;
  final EdgeInsetsGeometry? padding;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    return ClipRRect(
      borderRadius: BorderRadius.circular(22),
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 10, sigmaY: 10),
        child: Container(
          padding: padding ?? const EdgeInsets.all(14),
          decoration: BoxDecoration(
            color: dark
                ? Colors.white.withOpacity(0.07)
                : Colors.white.withOpacity(0.72),
            borderRadius: BorderRadius.circular(22),
            border: Border.all(
              color: dark
                  ? Colors.white.withOpacity(0.14)
                  : Colors.white.withOpacity(0.8),
            ),
            boxShadow: [
              BoxShadow(
                color: Colors.black.withOpacity(0.05),
                blurRadius: 12,
                offset: const Offset(0, 4),
              ),
            ],
          ),
          child: child,
        ),
      ),
    );
  }
}
