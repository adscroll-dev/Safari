import 'package:flutter/material.dart';

import 'theme.dart';

class OrbitDock extends StatelessWidget {
  const OrbitDock({
    super.key,
    required this.selectedIndex,
    required this.labels,
    required this.onSelected,
    required this.onHome,
    this.unreadCount = 0,
  });

  final int selectedIndex;
  final List<String> labels;
  final ValueChanged<int> onSelected;
  final VoidCallback onHome;
  final int unreadCount;

  static const _icons = [
    (Icons.chat_bubble_outline, Icons.chat_bubble),
    (Icons.circle_outlined, Icons.circle),
    (Icons.hub_outlined, Icons.hub),
    (Icons.call_outlined, Icons.call),
  ];

  @override
  Widget build(BuildContext context) {
    final reduceMotion =
        MediaQuery.maybeOf(context)?.disableAnimations ?? false;
    return SafeArea(
      top: false,
      child: Align(
        alignment: Alignment.bottomCenter,
        heightFactor: 1,
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 520),
          child: Container(
            margin: const EdgeInsetsDirectional.fromSTEB(16, 0, 16, 10),
            padding: const EdgeInsets.all(6),
            decoration: BoxDecoration(
              color: OrbitalColors.deepSpace.withOpacity(0.96),
              borderRadius: BorderRadius.circular(OrbitalRadii.dock),
              border: Border.all(color: Colors.white.withOpacity(0.10)),
              boxShadow: const [
                BoxShadow(
                  color: Color(0x66000000),
                  blurRadius: 28,
                  offset: Offset(0, 10),
                ),
              ],
            ),
            child: Row(
              children: [
                _tab(context, 0, reduceMotion),
                _tab(context, 1, reduceMotion),
                SizedBox(
                  width: 52,
                  height: 54,
                  child: Semantics(
                    button: true,
                    label: labels[0],
                    hint: 'Go to chats',
                    child: InkWell(
                      onTap: onHome,
                      customBorder: const CircleBorder(),
                      child: Container(
                        padding: const EdgeInsets.all(3),
                        decoration: const BoxDecoration(
                          shape: BoxShape.circle,
                          gradient: OrbitalColors.primaryOrbit,
                          boxShadow: [
                            BoxShadow(
                              color: Color(0xAA477BFF),
                              blurRadius: 18,
                              spreadRadius: 1,
                            ),
                            BoxShadow(
                              color: Color(0x8835D9F4),
                              blurRadius: 13,
                            ),
                          ],
                        ),
                        child: ClipOval(
                          child: Image.asset(
                            'assets/safari-logo-transparent.png',
                            fit: BoxFit.cover,
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
                _tab(context, 2, reduceMotion),
                _tab(context, 3, reduceMotion),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _tab(BuildContext context, int index, bool reduceMotion) {
    final selected = selectedIndex == index;
    return Expanded(
      child: Semantics(
        button: true,
        selected: selected,
        label: labels[index],
        child: InkWell(
          onTap: () => onSelected(index),
          borderRadius: BorderRadius.circular(18),
          child: AnimatedContainer(
            duration: reduceMotion
                ? Duration.zero
                : const Duration(milliseconds: 170),
            curve: Curves.easeOutCubic,
            constraints: const BoxConstraints(minHeight: 52),
            margin: const EdgeInsets.symmetric(horizontal: 1),
            padding: const EdgeInsets.symmetric(vertical: 4),
            decoration: BoxDecoration(
              color: selected
                  ? OrbitalColors.plasmaBlue.withOpacity(0.13)
                  : Colors.transparent,
              borderRadius: BorderRadius.circular(17),
              border: Border.all(
                color: selected
                    ? OrbitalColors.nebulaViolet.withOpacity(0.95)
                    : Colors.transparent,
                width: 1.4,
              ),
              boxShadow: selected
                  ? const [
                      BoxShadow(
                        color: Color(0x667B61FF),
                        blurRadius: 14,
                        spreadRadius: 1,
                      ),
                      BoxShadow(
                        color: Color(0x4435D9F4),
                        blurRadius: 10,
                      ),
                    ]
                  : const [],
            ),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                SizedBox(
                  width: 34,
                  height: 28,
                  child: Stack(
                    alignment: Alignment.center,
                    clipBehavior: Clip.none,
                    children: [
                      AnimatedScale(
                        duration: reduceMotion
                            ? Duration.zero
                            : const Duration(milliseconds: 160),
                        scale: selected ? 1.08 : 1,
                        child: Icon(
                          selected ? _icons[index].$2 : _icons[index].$1,
                          size: 20,
                          color: selected
                              ? OrbitalColors.electricCyan
                              : OrbitalColors.moonlight,
                        ),
                      ),
                      if (index == 0 && unreadCount > 0)
                        PositionedDirectional(
                          top: -1,
                          end: -1,
                          child: Container(
                            constraints: const BoxConstraints(
                              minWidth: 15,
                              minHeight: 15,
                            ),
                            padding: const EdgeInsets.symmetric(horizontal: 4),
                            decoration: BoxDecoration(
                              color: OrbitalColors.electricCyan,
                              borderRadius: BorderRadius.circular(10),
                            ),
                            alignment: Alignment.center,
                            child: Text(
                              unreadCount > 99 ? '99+' : '$unreadCount',
                              style: const TextStyle(
                                color: OrbitalColors.voidBlack,
                                fontSize: 9,
                                fontWeight: FontWeight.w700,
                              ),
                            ),
                          ),
                        ),
                    ],
                  ),
                ),
                Text(
                  labels[index],
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: OrbitalTypography.dockLabel.copyWith(
                    color: selected
                        ? OrbitalColors.stellarWhite
                        : OrbitalColors.moonlight,
                    fontWeight: selected ? FontWeight.w600 : FontWeight.w400,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
