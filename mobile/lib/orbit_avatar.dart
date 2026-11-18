import 'package:flutter/material.dart';

import 'theme.dart';

class OrbitAvatar extends StatelessWidget {
  const OrbitAvatar({
    super.key,
    required this.label,
    this.radius = 26,
  });

  final String label;
  final double radius;

  @override
  Widget build(BuildContext context) {
    // NEXUS AVATAR: a real gradient orbit ring around the avatar — the
    // ring is the Orbital signature, so it must read at a glance.
    const ringWidth = 2.0;
    return Container(
      padding: const EdgeInsets.all(ringWidth),
      decoration: const BoxDecoration(
        shape: BoxShape.circle,
        gradient: OrbitalColors.primaryOrbit,
      ),
      child: WaAvatar(
        label: label,
        radius: radius - ringWidth,
      ),
    );
  }
}
