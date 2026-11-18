import 'package:flutter/material.dart';

class OrbitShimmer extends StatefulWidget {
  const OrbitShimmer({super.key, required this.child});

  final Widget child;

  @override
  State<OrbitShimmer> createState() => _OrbitShimmerState();
}

class _OrbitShimmerState extends State<OrbitShimmer>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  bool? _motionEnabled;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 2400),
    );
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final enabled = !(MediaQuery.maybeOf(context)?.disableAnimations ?? false);
    if (_motionEnabled == enabled) return;
    _motionEnabled = enabled;
    if (enabled) {
      _controller.repeat();
    } else {
      _controller.stop();
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (_motionEnabled != true) return widget.child;
    final rtl = Directionality.of(context) == TextDirection.rtl;
    return AnimatedBuilder(
      animation: _controller,
      child: widget.child,
      builder: (context, child) {
        final progress = _controller.value * 3.2;
        final start = (rtl ? 1.6 : -1.6) + (rtl ? -progress : progress);
        return ShaderMask(
          blendMode: BlendMode.srcATop,
          shaderCallback: (bounds) => LinearGradient(
            begin: Alignment(start, 0),
            end: Alignment(start + (rtl ? -0.8 : 0.8), 0),
            colors: const [
              Color(0x00FFFFFF),
              Color(0x1AFFFFFF),
              Color(0x00FFFFFF),
            ],
            stops: const [0.35, 0.5, 0.65],
          ).createShader(bounds),
          child: child,
        );
      },
    );
  }
}
