import 'package:flutter/material.dart';

import '../features/coaching/preparation/preparation.dart';
import '../features/coaching/preparation/preparation_controller.dart';
import '../features/coaching/scene/scene_client.dart';
import 'app_routes.dart';

final class SpeakUpApp extends StatefulWidget {
  const SpeakUpApp({required this.sceneClient, super.key});

  final SceneClient sceneClient;

  @override
  State<SpeakUpApp> createState() => _SpeakUpAppState();
}

class _SpeakUpAppState extends State<SpeakUpApp> {
  late final PreparationController controller = PreparationController(client: widget.sceneClient);

  @override
  void dispose() {
    controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => MaterialApp(
        title: 'SpeakUp',
        theme: ThemeData(colorSchemeSeed: Colors.indigo, useMaterial3: true),
        initialRoute: AppRoutes.practice,
        routes: {AppRoutes.practice: (_) => PreparationPage(controller: controller)},
      );
}
