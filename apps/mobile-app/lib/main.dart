// Entry point — scaffold only (Phase 1). Real screens (glanceable
// notification-driven home, match replays, season stats) land per
// docs/phases.md, following docs/design.md Section 12.
import 'package:flutter/material.dart';

void main() {
  runApp(const CricketDrsApp());
}

class CricketDrsApp extends StatelessWidget {
  const CricketDrsApp({super.key});

  @override
  Widget build(BuildContext context) {
    return const MaterialApp(
      title: 'Cricket DRS',
      home: Scaffold(
        body: Center(child: Text('Cricket DRS — Mobile (scaffold)')),
      ),
    );
  }
}
