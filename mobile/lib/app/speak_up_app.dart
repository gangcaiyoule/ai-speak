import 'package:flutter/material.dart';

import '../features/coaching/preparation/preparation.dart';
import '../features/coaching/preparation/preparation_controller.dart';
import '../features/coaching/practice/practice.dart';
import '../features/coaching/practice/practice_controller.dart';
import '../features/coaching/coaching_clients.dart';
import '../features/coaching/practice_plan/practice_plan_client.dart';
import '../features/coaching/scene/scene_client.dart';
import '../features/voice_stream/voice_debug_page.dart';
import '../identity/auth_gate.dart';
import '../identity/identity_client.dart';
import 'app_routes.dart';

final class SpeakUpApp extends StatefulWidget {
  const SpeakUpApp({required this.sceneClient, required this.identityClient, this.planClient, this.practiceClient, super.key});

  final SceneClient sceneClient;
  final IdentityClient identityClient;
  final PracticePlanClient? planClient;
  final PracticeClient? practiceClient;

  @override
  State<SpeakUpApp> createState() => _SpeakUpAppState();
}

class _SpeakUpAppState extends State<SpeakUpApp> {
  late final PreparationController controller = PreparationController(client: widget.sceneClient, planClient: widget.planClient);
  PracticeController? _practiceController;

  @override
  void dispose() {
    controller.dispose();
    _practiceController?.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => MaterialApp(
        title: 'SpeakUp',
        theme: ThemeData(colorSchemeSeed: Colors.indigo, useMaterial3: true),
        home: AuthGate(identityClient: widget.identityClient, authenticatedBuilder: (_) => _practiceController == null ? PreparationPage(controller: controller, onStartSession: _startSession) : PracticePage(controller: _practiceController!)),
        routes: {
          AppRoutes.voiceDebug: (_) => const VoiceDebugPage(),
        },
      );

  Future<void> _startSession(PracticePlan plan) async {
    final client = widget.practiceClient;
    if (client == null) return;
    try {
      final draft = await client.createSession(plan.id);
      final active = await client.activateSession(draft.id);
      if (!mounted) return;
      setState(() => _practiceController = PracticeController(client: client, sessionID: active.id, initialSession: active));
    } on Exception catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(error.toString())));
    }
  }
}
