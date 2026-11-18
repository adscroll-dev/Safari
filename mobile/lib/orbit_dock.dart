import 'package:flutter/material.dart';

import 'theme.dart';

class OrbitDock extends StatelessWidget {
  const OrbitDock({
    super.key,
    required this.selectedIndex,
    required this.labels,
    required this.onSelected,
    this.unreadCount = 0,
  });

  final int selectedIndex;
  final List<String> labels;
  final ValueChanged<int> onSelected;
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
    // NOTE: no Center/Align here — as a Scaffold bottomNavigationBar this
    // widget must size to its content. A Center expands to the full
    // remaining height, which starves the Scaffold body to zero height
    // (the whole tab content collapses and the dock floats mid-screen).
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
              color: OrbitalColors.deepSpace.withOpacity(0.94),
              borderRadius: BorderRadius.circular(OrbitalRadii.dock),
              border: Border.all(color: Colors.white.withOpacity(0.08)),
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
                for (var index = 0; index < _icons.length; index++)
                  Expanded(
                    child: Semantics(
                      button: true,
                      selected: selectedIndex == index,
                      label: labels[index],
                      child: InkWell(
                        onTap: () => onSelected(index),
                        borderRadius: BorderRadius.circular(24),
                        child: AnimatedContainer(
                          duration: reduceMotion
                              ? Duration.zero
                              : const Duration(milliseconds: 160),
                          curve: Curves.easeOutCubic,
                          constraints: const BoxConstraints(minHeight: 52),
                          padding: const EdgeInsets.symmetric(vertical: 5),
                          child: Column(
                            mainAxisSize: MainAxisSize.min,
                            mainAxisAlignment: MainAxisAlignment.center,
                            children: [
                              SizedBox(
                                width: 36,
                                height: 30,
                                child: Stack(
                                  alignment: Alignment.center,
                                  clipBehavior: Clip.none,
                                  children: [
                                    if (selectedIndex == index)
                                      const Positioned.fill(
                                        child: CustomPaint(
                                          painter: _OrbitArcPainter(),
                                        ),
                                      ),
                                    AnimatedScale(
                                      duration: reduceMotion
                                          ? Duration.zero
                                          : const Duration(milliseconds: 160),
                                      scale: selectedIndex == index ? 1.08 : 1,
                                      curve: Curves.easeOutCubic,
                                      child: Icon(
                                        selectedIndex == index
                                            ? _icons[index].$2
                                            : _icons[index].$1,
                                        size: 21,
                                        color: selectedIndex == index
                                            ? OrbitalColors.electricCyan
                                            : OrbitalColors.mutedBlue,
                                      ),
                                    ),
                                    if (index == 0 && unreadCount > 0)
                                      PositionedDirectional(
                                        top: -1,
                                        end: 0,
                                        child: Container(
                                          constraints: const BoxConstraints(
                                            minWidth: 15,
                                            minHeight: 15,
                                          ),
                                          padding: const EdgeInsets.symmetric(
                                              horizontal: 4),
                                          decoration: BoxDecoration(
                                            color: OrbitalColors.electricCyan,
                                            borderRadius:
                                                BorderRadius.circular(10),
                                          ),
                                          alignment: Alignment.center,
                                          child: Text(
                                            unreadCount > 99
                                                ? '99+'
                                                : '$unreadCount',
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
                              const SizedBox(height: 1),
                              Text(
                                labels[index],
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: OrbitalTypography.dockLabel.copyWith(
                                  color: selectedIndex == index
                                      ? OrbitalColors.stellarWhite
                                      : OrbitalColors.mutedBlue,
                                  fontWeight: selectedIndex == index
                                      ? FontWeight.w600
                                      : FontWeight.w400,
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
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

class _OrbitArcPainter extends CustomPainter {
  const _OrbitArcPainter();

  @override
  void paint(Canvas canvas, Size size) {
    final rect = Rect.fromCenter(
      center: Offset(size.width / 2, size.height / 2),
      width: size.width * 0.94,
      height: size.height * 0.62,
    );
    // A soft glowing ellipse BEHIND the active icon — the dedicated
    // orbital indicator for the active tab — then the crisp orbit arc
    // around it. The active tab must read as a glowing orbit, not a
    // hairline.
    final haloPaint = Paint()
      ..color = OrbitalColors.electricCyan.withOpacity(0.20)
      ..style = PaintingStyle.fill;
    canvas.drawOval(rect, haloPaint);
    final glowPaint = Paint()
      ..color = OrbitalColors.electricCyan.withOpacity(0.28)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 5
      ..strokeCap = StrokeCap.round;
    canvas.drawArc(rect, -0.5, 5.3, false, glowPaint);
    final paint = Paint()
      // NOTE: createShader needs an explicit TextDirection here — paint()
      // has no Directionality, and primaryOrbit uses AlignmentDirectional
      // begin/end, which throws "Cannot resolve AlignmentDirectional
      // without a TextDirection" when resolved with a null direction.
      ..shader = OrbitalColors.primaryOrbit.createShader(
        rect,
        textDirection: TextDirection.ltr,
      )
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2.5
      ..strokeCap = StrokeCap.round;
    canvas.drawArc(rect, -0.5, 5.3, false, paint);
  }

  @override
  bool shouldRepaint(_OrbitArcPainter oldDelegate) => false;
}
