import 'package:flutter/material.dart';

/// Starts the minimal ai-speak Flutter application.
void main() {
  runApp(const AiSpeakApp());
}

/// Root widget for the ai-speak client.
class AiSpeakApp extends StatelessWidget {
  /// Creates the root application widget.
  const AiSpeakApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'AI Speak',
      theme: ThemeData(colorSchemeSeed: Colors.indigo),
      home: const Scaffold(
        body: Center(child: Text('AI Speak')),
      ),
    );
  }
}
