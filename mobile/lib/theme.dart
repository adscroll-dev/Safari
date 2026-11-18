// Safari theme — premium futuristic (v0.7.3).
//
// Safari's own identity: deep midnight navy, electric cyan +
// turquoise accents, subtle purple highlights, glassmorphism,
// soft neon glows. Dark mode is the star (midnight navy, never
// pure black); light mode is a clean pale-sky companion.
// The Safari planet logo and name stay Safari; the layout, colors
// and shapes are Safari's own — not a WhatsApp clone.
import 'package:flutter/material.dart';

class OrbitalColors {
  static const voidBlack = Color(0xFF050812);
  static const deepSpace = Color(0xFF080F20);
  static const nebulaNavy = Color(0xFF101B32);
  static const orbitalSurface = Color(0xFF14223B);
  static const elevatedSurface = Color(0xFF1A2C49);
  static const electricCyan = Color(0xFF35D9F4);
  static const plasmaBlue = Color(0xFF477BFF);
  static const nebulaViolet = Color(0xFF9877FF);
  static const auroraMint = Color(0xFF55E6BC);
  static const solarAmber = Color(0xFFFFBF69);
  static const signalCoral = Color(0xFFFF637D);
  static const stellarWhite = Color(0xFFF3F7FF);
  static const moonlight = Color(0xFFB8C5D9);
  static const mutedBlue = Color(0xFF7889A5);
  static const disabled = Color(0xFF52617A);

  static const primaryOrbit = LinearGradient(
    colors: [electricCyan, plasmaBlue],
    begin: AlignmentDirectional.centerStart,
    end: AlignmentDirectional.centerEnd,
  );
  static const nebulaFlow = LinearGradient(
    colors: [plasmaBlue, nebulaViolet],
    begin: AlignmentDirectional.centerStart,
    end: AlignmentDirectional.centerEnd,
  );
  static const auroraGlow = LinearGradient(
    colors: [electricCyan, auroraMint],
    begin: AlignmentDirectional.centerStart,
    end: AlignmentDirectional.centerEnd,
  );
}

class OrbitalSpacing {
  static const xs = 4.0;
  static const sm = 8.0;
  static const md = 12.0;
  static const lg = 16.0;
  static const xl = 20.0;
  static const xxl = 24.0;
  static const section = 32.0;
  static const largeSection = 40.0;
}

class OrbitalRadii {
  static const smallButton = 14.0;
  static const mainButton = 18.0;
  static const card = 22.0;
  static const floatingPanel = 26.0;
  static const dock = 30.0;
  static const bottomSheet = 28.0;
}

class OrbitalTypography {
  static const pageTitle = TextStyle(
    fontSize: 28,
    fontWeight: FontWeight.w700,
    height: 1.25,
  );
  static const sectionTitle = TextStyle(
    fontSize: 21,
    fontWeight: FontWeight.w600,
    height: 1.3,
  );
  static const cardTitle = TextStyle(
    fontSize: 17,
    fontWeight: FontWeight.w600,
    height: 1.35,
  );
  static const body = TextStyle(
    fontSize: 15,
    fontWeight: FontWeight.w400,
    height: 1.45,
  );
  static const button = TextStyle(
    fontSize: 15,
    fontWeight: FontWeight.w600,
  );
  static const secondary = TextStyle(
    fontSize: 13,
    fontWeight: FontWeight.w400,
    height: 1.4,
  );
  static const caption = TextStyle(
    fontSize: 11,
    fontWeight: FontWeight.w400,
    height: 1.35,
  );
  static const dockLabel = TextStyle(
    fontSize: 10,
    fontWeight: FontWeight.w400,
    height: 1.3,
  );

  static const textTheme = TextTheme(
    titleLarge: pageTitle,
    titleMedium: sectionTitle,
    titleSmall: cardTitle,
    bodyLarge: body,
    bodyMedium: body,
    bodySmall: secondary,
    labelSmall: caption,
  );
}

class VyroColors {
  // Brand accents (kept under the old names so existing widgets
  // keep working — they now render in Safari cyan/purple).
  static const Color blue = OrbitalColors.electricCyan;
  static const Color purple = OrbitalColors.nebulaViolet;
  static const Color blueDeep = OrbitalColors.plasmaBlue;
  static const Color purpleDeep = Color(0xFF7056D8);
  static const Color electricBlue = OrbitalColors.plasmaBlue;
  static const Color midnight = OrbitalColors.voidBlack;

  static const Color teal = OrbitalColors.auroraMint;
  static const Color tealDeep = Color(0xFF2EBB98);
  static const Color green = OrbitalColors.auroraMint;
  static const Color greenDark = OrbitalColors.auroraMint;
  static const Color readBlue = OrbitalColors.electricCyan;

  /// Cyan -> blue -> purple: the Safari brand gradient.
  static const LinearGradient brandGradient = LinearGradient(
    colors: [OrbitalColors.electricCyan, OrbitalColors.plasmaBlue],
    begin: AlignmentDirectional.centerStart,
    end: AlignmentDirectional.centerEnd,
  );

  static const LinearGradient brandGradientDark = LinearGradient(
    colors: [
      OrbitalColors.electricCyan,
      OrbitalColors.plasmaBlue,
    ],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );

  /// Story ring: cyan -> purple, used as a border around avatars
  /// with unviewed status updates.
  static const LinearGradient storyRing = LinearGradient(
    colors: [OrbitalColors.electricCyan, OrbitalColors.nebulaViolet],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );

  /// Soft neon glow used behind hero icons and empty states.
  static const Color glowCyan = Color(0x3335D9F4);
  static const Color glowPurple = Color(0x229877FF);

  static const LinearGradient heroGradient = LinearGradient(
    colors: [
      Color(0xEE101B32),
      Color(0xEE14223B),
      Color(0xEE1A2C49),
    ],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );
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
  static const Color darkHeader = OrbitalColors.deepSpace;
  static const Color darkBackground = OrbitalColors.voidBlack;
  static const Color darkChatBackground = OrbitalColors.voidBlack;
  static const Color darkBubbleOut = Color(0xFF17335A);
  static const Color darkBubbleIn = OrbitalColors.orbitalSurface;
  static const Color darkText = OrbitalColors.stellarWhite;
  static const Color darkTextSecondary = OrbitalColors.moonlight;
  static const Color darkDivider = Color(0x1FFFFFFF);
  static const Color darkInputBackground = OrbitalColors.nebulaNavy;
  static const Color darkIcon = OrbitalColors.moonlight;
  // Shared.
  static const Color accent = OrbitalColors.electricCyan;
  static const Color accentDark = OrbitalColors.plasmaBlue;
  static const Color readBlue = OrbitalColors.electricCyan;
  static const Color unreadBadge = OrbitalColors.electricCyan;
  static const Color fabLight = OrbitalColors.plasmaBlue;
  static const Color fabDark = OrbitalColors.plasmaBlue;
  // Extra surfaces for glass cards.
  static const Color darkSurface = OrbitalColors.nebulaNavy;
  static const Color darkSurface2 = OrbitalColors.orbitalSurface;
  static const Color darkBorder = Color(0x1FFFFFFF);
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
    fontFamily: 'Manrope',
    fontFamilyFallback: const ['Vazirmatn', 'Roboto'],
    textTheme: OrbitalTypography.textTheme.apply(
      bodyColor: WaColors.lightText,
      displayColor: WaColors.lightText,
    ),
    appBarTheme: const AppBarTheme(
      backgroundColor: WaColors.lightHeader,
      foregroundColor: Colors.white,
      elevation: 0,
      scrolledUnderElevation: 0,
      centerTitle: false,
      titleTextStyle: TextStyle(
        fontSize: 21,
        fontWeight: FontWeight.w600,
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
        borderRadius: BorderRadius.circular(OrbitalRadii.mainButton),
        borderSide: const BorderSide(color: Color(0xFFE2E8F0)),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(OrbitalRadii.mainButton),
        borderSide: const BorderSide(color: Color(0xFFE2E8F0)),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(OrbitalRadii.mainButton),
        borderSide: const BorderSide(color: Color(0xFF0891B2), width: 1.6),
      ),
    ),
    cardTheme: CardTheme(
      elevation: 0,
      color: WaColors.lightSurface,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(OrbitalRadii.card),
        side: const BorderSide(color: WaColors.lightBorder),
      ),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: const Color(0xFF0891B2),
        foregroundColor: Colors.white,
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
        minimumSize: const Size(0, 52),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(OrbitalRadii.mainButton),
        ),
        textStyle: OrbitalTypography.button,
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
    seedColor: OrbitalColors.electricCyan,
    brightness: Brightness.dark,
  ).copyWith(
    primary: OrbitalColors.electricCyan,
    secondary: OrbitalColors.nebulaViolet,
    surface: OrbitalColors.nebulaNavy,
    error: OrbitalColors.signalCoral,
    onPrimary: OrbitalColors.voidBlack,
    onSecondary: OrbitalColors.stellarWhite,
    onSurface: OrbitalColors.stellarWhite,
    onError: OrbitalColors.voidBlack,
  );
  return ThemeData(
    colorScheme: scheme,
    useMaterial3: true,
    scaffoldBackgroundColor: WaColors.darkBackground,
    fontFamily: 'Manrope',
    fontFamilyFallback: const ['Vazirmatn', 'Roboto'],
    textTheme: OrbitalTypography.textTheme.apply(
      bodyColor: OrbitalColors.stellarWhite,
      displayColor: OrbitalColors.stellarWhite,
    ),
    appBarTheme: const AppBarTheme(
      backgroundColor: WaColors.darkHeader,
      foregroundColor: Colors.white,
      elevation: 0,
      scrolledUnderElevation: 0,
      centerTitle: false,
      titleTextStyle: TextStyle(
        fontSize: 21,
        fontWeight: FontWeight.w600,
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
        borderRadius: BorderRadius.circular(OrbitalRadii.mainButton),
        borderSide: const BorderSide(color: Color(0xFF1C2A44)),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(OrbitalRadii.mainButton),
        borderSide: const BorderSide(color: Color(0xFF1C2A44)),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(OrbitalRadii.mainButton),
        borderSide: const BorderSide(color: WaColors.accent, width: 1.6),
      ),
    ),
    cardTheme: CardTheme(
      elevation: 0,
      color: WaColors.darkSurface,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(OrbitalRadii.card),
        side: const BorderSide(color: WaColors.darkBorder),
      ),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: WaColors.accentDark,
        foregroundColor: Colors.white,
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
        minimumSize: const Size(0, 52),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(OrbitalRadii.mainButton),
        ),
        textStyle: OrbitalTypography.button,
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
            color: Color(0x4422D3EE),
            blurRadius: 11,
            offset: Offset(0, 4),
          ),
          BoxShadow(
            color: Color(0x55202A52),
            blurRadius: 4,
            offset: Offset(0, 2),
          ),
        ],
      ),
      alignment: Alignment.center,
      child: Stack(
        alignment: Alignment.center,
        children: [
          Positioned(
            top: 1,
            left: radius * 0.2,
            right: radius * 0.2,
            height: radius * 0.62,
            child: DecoratedBox(
              decoration: BoxDecoration(
                borderRadius: BorderRadius.circular(radius),
                gradient: const LinearGradient(
                  colors: [Color(0x55FFFFFF), Color(0x00FFFFFF)],
                  begin: Alignment.topCenter,
                  end: Alignment.bottomCenter,
                ),
              ),
            ),
          ),
          Text(
            initial,
            style: TextStyle(
              color: Colors.white,
              fontWeight: FontWeight.bold,
              fontSize: radius * 0.75,
              shadows: const [
                Shadow(
                  color: Color(0x44301E70),
                  blurRadius: 5,
                  offset: Offset(0, 1),
                ),
              ],
            ),
          ),
        ],
      ),
    );
    if (!ring) return avatar;
    return Container(
      padding: const EdgeInsets.all(2.5),
      decoration: const BoxDecoration(
        gradient: VyroColors.storyRing,
        shape: BoxShape.circle,
        boxShadow: [
          BoxShadow(
            color: Color(0x3322D3EE),
            blurRadius: 11,
            offset: Offset(0, 3),
          ),
        ],
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
    final borderGradient = LinearGradient(
      colors: dark
          ? const [Color(0x9940DDF4), Color(0x667A62EC), Color(0x5540DDF4)]
          : const [Color(0xBB40CBE7), Color(0x775B78E8), Color(0x663FC9D8)],
      begin: Alignment.topLeft,
      end: Alignment.bottomRight,
    );
    final surfaceGradient = LinearGradient(
      colors: dark
          ? const [Color(0xF51D3552), Color(0xF20D192D), Color(0xF7091425)]
          : const [Color(0xFFFFFFFF), Color(0xFBF9FCFF), Color(0xF2EDF4FA)],
      begin: Alignment.topCenter,
      end: Alignment.bottomRight,
    );
    final card = Container(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(radius),
        gradient: borderGradient,
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(dark ? 0.40 : 0.10),
            blurRadius: 24,
            offset: const Offset(0, 10),
          ),
          BoxShadow(
            color: VyroColors.glowCyan.withOpacity(dark ? 0.32 : 0.16),
            blurRadius: 22,
            offset: const Offset(0, 3),
          ),
          BoxShadow(
            color: VyroColors.glowPurple.withOpacity(dark ? 0.16 : 0.08),
            blurRadius: 30,
            offset: const Offset(0, 12),
          ),
        ],
      ),
      padding: const EdgeInsets.all(1),
      child: Material(
        color: Colors.transparent,
        child: Ink(
          decoration: BoxDecoration(
            gradient: surfaceGradient,
            borderRadius: BorderRadius.circular(radius - 1),
            border: Border.all(
              color: Colors.white.withOpacity(dark ? 0.08 : 0.75),
            ),
          ),
          child: InkWell(
            onTap: onTap,
            borderRadius: BorderRadius.circular(radius - 1),
            child: Stack(
              children: [
                Positioned(
                  top: 1,
                  left: 1,
                  right: 1,
                  height: 58,
                  child: IgnorePointer(
                    child: DecoratedBox(
                      decoration: BoxDecoration(
                        borderRadius: BorderRadius.vertical(
                          top: Radius.circular(radius - 1),
                        ),
                        gradient: LinearGradient(
                          colors: [
                            Colors.white.withOpacity(dark ? 0.10 : 0.58),
                            Colors.white.withOpacity(dark ? 0.025 : 0.12),
                            Colors.transparent,
                          ],
                          begin: Alignment.topCenter,
                          end: Alignment.bottomCenter,
                        ),
                      ),
                    ),
                  ),
                ),
                Positioned(
                  top: 0,
                  left: radius * 0.8,
                  right: radius * 0.8,
                  height: 1.5,
                  child: IgnorePointer(
                    child: DecoratedBox(
                      decoration: BoxDecoration(
                        gradient: LinearGradient(
                          colors: [
                            Colors.transparent,
                            Colors.white.withOpacity(dark ? 0.34 : 0.95),
                            Colors.transparent,
                          ],
                        ),
                      ),
                    ),
                  ),
                ),
                Padding(
                  padding: padding ?? const EdgeInsets.all(16),
                  child: child,
                ),
              ],
            ),
          ),
        ),
      ),
    );
    return card;
  }
}

/// A compact animated filter chip for the main tab lists.
class SafariFilterChip extends StatelessWidget {
  const SafariFilterChip({
    super.key,
    required this.label,
    required this.selected,
    required this.onTap,
    this.onLongPress,
  });

  final String label;
  final bool selected;
  final VoidCallback onTap;
  final VoidCallback? onLongPress;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    return AnimatedContainer(
      duration: const Duration(milliseconds: 160),
      curve: Curves.easeOutCubic,
      decoration: BoxDecoration(
        color: selected
            ? OrbitalColors.elevatedSurface
            : dark
                ? OrbitalColors.nebulaNavy.withOpacity(0.88)
                : Colors.white.withOpacity(0.94),
        borderRadius: BorderRadius.circular(18),
        border: Border.all(
          color: selected
              ? OrbitalColors.electricCyan.withOpacity(0.45)
              : dark
                  ? WaColors.darkBorder
                  : WaColors.lightBorder,
        ),
        boxShadow: selected
            ? const [
                BoxShadow(
                  color: Color(0x2235D9F4),
                  blurRadius: 10,
                  offset: Offset(0, 2),
                ),
              ]
            : null,
      ),
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          borderRadius: BorderRadius.circular(18),
          onTap: onTap,
          onLongPress: onLongPress,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 15, vertical: 9),
            child: Text(
              label,
              style: TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w600,
                color: selected
                    ? OrbitalColors.stellarWhite
                    : dark
                        ? WaColors.darkText
                        : WaColors.lightText,
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// Dark-glass hero action with a luminous icon tile and a short description.
class SafariHeroCard extends StatelessWidget {
  const SafariHeroCard({
    super.key,
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.onTap,
    this.actionLabel,
  });

  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback onTap;
  final String? actionLabel;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final secondary =
        dark ? WaColors.darkTextSecondary : WaColors.lightTextSecondary;
    return SafariGlassCard(
      onTap: onTap,
      padding: const EdgeInsets.all(16),
      child: Row(
        children: [
          Container(
            width: 52,
            height: 52,
            decoration: BoxDecoration(
              gradient: VyroColors.brandGradient,
              borderRadius: BorderRadius.circular(16),
              boxShadow: const [
                BoxShadow(
                  color: Color(0x5522D3EE),
                  blurRadius: 18,
                  offset: Offset(0, 6),
                ),
                BoxShadow(
                  color: Color(0x338B5CF6),
                  blurRadius: 22,
                  offset: Offset(0, 3),
                ),
              ],
            ),
            child: Stack(
              alignment: Alignment.center,
              children: [
                Positioned(
                  top: 1,
                  left: 5,
                  right: 5,
                  height: 24,
                  child: DecoratedBox(
                    decoration: BoxDecoration(
                      borderRadius: BorderRadius.circular(14),
                      gradient: const LinearGradient(
                        colors: [Color(0x66FFFFFF), Color(0x00FFFFFF)],
                        begin: Alignment.topCenter,
                        end: Alignment.bottomCenter,
                      ),
                    ),
                  ),
                ),
                Icon(icon, color: Colors.white, size: 27),
              ],
            ),
          ),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: const TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 3),
                Text(
                  subtitle,
                  style: TextStyle(fontSize: 13, color: secondary),
                ),
                if (actionLabel != null) ...[
                  const SizedBox(height: 8),
                  Text(
                    actionLabel!,
                    style: const TextStyle(
                      color: WaColors.accent,
                      fontSize: 12,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ],
              ],
            ),
          ),
          Icon(Icons.chevron_right, color: secondary),
        ],
      ),
    );
  }
}

/// Gradient call action used for voice and video call entry points.
class SafariGradientButton extends StatelessWidget {
  const SafariGradientButton({
    super.key,
    required this.icon,
    required this.label,
    required this.onPressed,
  });

  final IconData icon;
  final String label;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onPressed,
        borderRadius: BorderRadius.circular(16),
        child: Ink(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 13),
          decoration: BoxDecoration(
            gradient: VyroColors.brandGradient,
            borderRadius: BorderRadius.circular(16),
            boxShadow: const [
              BoxShadow(
                color: Color(0x4422D3EE),
                blurRadius: 16,
                offset: Offset(0, 6),
              ),
              BoxShadow(
                color: Color(0x338B5CF6),
                blurRadius: 22,
                offset: Offset(0, 4),
              ),
            ],
            border: Border.all(color: const Color(0x66FFFFFF)),
          ),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(icon, color: Colors.white, size: 19),
              const SizedBox(width: 7),
              Flexible(
                child: Text(
                  label,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    color: Colors.white,
                    fontSize: 14,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
            ],
          ),
        ),
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
    this.compact = false,
  });

  final IconData icon;
  final String title;
  final String subtitle;
  final String? buttonLabel;
  final VoidCallback? onPressed;

  /// Optional custom emblem (e.g. the planet logo). When null, a
  /// gradient circle with [icon] is used.
  final Widget? emblem;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final secondary =
        dark ? WaColors.darkTextSecondary : WaColors.lightTextSecondary;
    final action = onPressed;
    final actionText = buttonLabel;
    return Center(
      child: SingleChildScrollView(
        padding:
            EdgeInsets.fromLTRB(32, compact ? 12 : 40, 32, compact ? 12 : 40),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            emblem ??
                Container(
                  width: compact ? 78 : 96,
                  height: compact ? 78 : 96,
                  padding: const EdgeInsets.all(1.5),
                  decoration: BoxDecoration(
                    gradient: const LinearGradient(
                      colors: [
                        Color(0xCCFFFFFF),
                        Color(0x7722D3EE),
                        Color(0x998B5CF6),
                      ],
                      begin: Alignment.topLeft,
                      end: Alignment.bottomRight,
                    ),
                    shape: BoxShape.circle,
                    boxShadow: [
                      BoxShadow(
                        color: VyroColors.glowCyan.withOpacity(0.75),
                        blurRadius: 30,
                        offset: const Offset(0, 8),
                      ),
                      BoxShadow(
                        color: VyroColors.glowPurple.withOpacity(0.55),
                        blurRadius: 48,
                        offset: const Offset(0, 12),
                      ),
                      BoxShadow(
                        color: Colors.black.withOpacity(dark ? 0.28 : 0.08),
                        blurRadius: 10,
                        offset: const Offset(0, 5),
                      ),
                    ],
                  ),
                  child: Container(
                    decoration: const BoxDecoration(
                      gradient: VyroColors.brandGradient,
                      shape: BoxShape.circle,
                    ),
                    child: Stack(
                      alignment: Alignment.center,
                      children: [
                        Positioned(
                          top: 2,
                          left: 12,
                          right: 12,
                          height: compact ? 30 : 38,
                          child: const DecoratedBox(
                            decoration: BoxDecoration(
                              borderRadius: BorderRadius.vertical(
                                top: Radius.circular(50),
                              ),
                              gradient: LinearGradient(
                                colors: [
                                  Color(0x66FFFFFF),
                                  Color(0x00FFFFFF),
                                ],
                                begin: Alignment.topCenter,
                                end: Alignment.bottomCenter,
                              ),
                            ),
                          ),
                        ),
                        Icon(
                          icon,
                          size: compact ? 36 : 44,
                          color: Colors.white,
                        ),
                      ],
                    ),
                  ),
                ),
            SizedBox(height: compact ? 14 : 24),
            Text(
              title,
              textAlign: TextAlign.center,
              style: TextStyle(
                fontSize: compact ? 18 : 19,
                fontWeight: FontWeight.w800,
                color: dark ? WaColors.darkText : WaColors.lightText,
                letterSpacing: 0.1,
              ),
            ),
            SizedBox(height: compact ? 6 : 8),
            Text(
              subtitle,
              textAlign: TextAlign.center,
              style: TextStyle(fontSize: 14, height: 1.4, color: secondary),
            ),
            if (actionText != null && action != null) ...[
              SizedBox(height: compact ? 14 : 24),
              SafariGradientButton(
                icon: Icons.add,
                label: actionText,
                onPressed: action,
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class NexusEmptyOrbit extends StatelessWidget {
  const NexusEmptyOrbit({super.key});

  @override
  Widget build(BuildContext context) {
    return ExcludeSemantics(
      child: SizedBox(
        width: 132,
        height: 104,
        child: CustomPaint(
          painter: _OpenOrbitPainter(
            OrbitalColors.electricCyan.withOpacity(0.52),
          ),
          child: Center(
            child: Container(
              width: 20,
              height: 20,
              decoration: const BoxDecoration(
                shape: BoxShape.circle,
                gradient: OrbitalColors.primaryOrbit,
                boxShadow: [
                  BoxShadow(
                    color: Color(0x4435D9F4),
                    blurRadius: 14,
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _OpenOrbitPainter extends CustomPainter {
  const _OpenOrbitPainter(this.color);

  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1.2
      ..strokeCap = StrokeCap.round;
    final rect = Rect.fromCenter(
      center: Offset(size.width / 2, size.height / 2),
      width: size.width * 0.82,
      height: size.height * 0.55,
    );
    canvas.drawArc(rect, -0.18, 5.18, false, paint);
  }

  @override
  bool shouldRepaint(_OpenOrbitPainter oldDelegate) =>
      oldDelegate.color != color;
}

/// A restrained spatial field behind independently scrolling sections.
class SafariAuroraBackdrop extends StatelessWidget {
  const SafariAuroraBackdrop({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    return Stack(
      fit: StackFit.expand,
      children: [
        AnimatedContainer(
          duration: const Duration(milliseconds: 260),
          curve: Curves.easeOutCubic,
          color: Theme.of(context).scaffoldBackgroundColor,
        ),
        IgnorePointer(
          child: CustomPaint(
            painter: _OrbitalBackgroundPainter(
              // Visible orbit lines — the ORBITAL signature. Strong enough
              // to read at a glance, light enough not to fight the list.
              // In light mode the rings are calmer so the content stays
              // primary (per the ChatGPT design review).
              dark
                  ? OrbitalColors.electricCyan.withOpacity(0.13)
                  : OrbitalColors.plasmaBlue.withOpacity(0.10),
              dark
                  ? OrbitalColors.nebulaViolet.withOpacity(0.11)
                  : OrbitalColors.nebulaViolet.withOpacity(0.08),
            ),
          ),
        ),
        child,
      ],
    );
  }
}

class _OrbitalBackgroundPainter extends CustomPainter {
  const _OrbitalBackgroundPainter(this.color, this.accent);

  final Color color;
  final Color accent;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1.5;
    final center = Offset(size.width * 0.78, size.height * 0.2);
    for (final scale in [1.0, 1.34, 1.7]) {
      canvas.drawOval(
        Rect.fromCenter(
          center: center,
          width: size.width * 0.78 * scale,
          height: size.width * 0.32 * scale,
        ),
        paint,
      );
    }
    // A second, violet orbit crossing from the lower-left, so the
    // background reads as a real orbital field, not a plain tint.
    final accentPaint = Paint()
      ..color = accent
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1.5;
    canvas.drawOval(
      Rect.fromCenter(
        center: Offset(size.width * 0.12, size.height * 0.86),
        width: size.width * 0.9,
        height: size.width * 0.38,
      ),
      accentPaint,
    );
  }

  @override
  bool shouldRepaint(_OrbitalBackgroundPainter oldDelegate) =>
      oldDelegate.color != color || oldDelegate.accent != accent;
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
