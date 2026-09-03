import 'package:flutter/material.dart';

import '../features/coaching/preparation/preparation.dart';
import '../features/coaching/preparation/preparation_controller.dart';
import '../features/coaching/scene/scene_client.dart';
import '../identity/auth_gate.dart';
import '../identity/identity_client.dart';
import 'app_routes.dart';

final class SpeakUpApp extends StatefulWidget {
  const SpeakUpApp({required this.sceneClient, required this.identityClient, super.key});

  final SceneClient sceneClient;
  final IdentityClient identityClient;

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
        home: AuthGate(identityClient: widget.identityClient, authenticatedBuilder: (_) => PreparationPage(controller: controller)),
      );
}
