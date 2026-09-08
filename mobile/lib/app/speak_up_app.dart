import 'package:flutter/material.dart';

import '../features/coaching/preparation/preparation.dart';
import '../features/coaching/preparation/preparation_controller.dart';
import '../features/coaching/practice/practice.dart';
import '../features/coaching/practice/practice_controller.dart';
import '../features/coaching/coaching_clients.dart' hide SceneClient;
import '../features/coaching/practice_plan/practice_plan.dart' show PracticePlan;
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
  @override
  Widget build(BuildContext context) => MaterialApp(
        title: 'SpeakUp',
        theme: ThemeData(colorSchemeSeed: Colors.indigo, useMaterial3: true),
        home: AuthGate(identityClient: widget.identityClient, authenticatedBuilder: (_) => _PracticeRecoveryHome(sceneClient: widget.sceneClient, planClient: widget.planClient, practiceClient: widget.practiceClient)),
        routes: {
          AppRoutes.voiceDebug: (_) => const VoiceDebugPage(),
        },
      );
}

final class _PracticeRecoveryHome extends StatefulWidget {
  const _PracticeRecoveryHome({required this.sceneClient, this.planClient, this.practiceClient});
  final SceneClient sceneClient;
  final PracticePlanClient? planClient;
  final PracticeClient? practiceClient;

  @override
  State<_PracticeRecoveryHome> createState() => _PracticeRecoveryHomeState();
}

class _PracticeRecoveryHomeState extends State<_PracticeRecoveryHome> {
  late final PreparationController _preparation = PreparationController(client: widget.sceneClient, planClient: widget.planClient);
  PracticeController? _practiceController;
  PracticeSession? _draftSession;
  Object? _recoveryError;
  var _loading = true;

  @override
  void initState() {
    super.initState();
    _restore();
  }

  @override
  void dispose() {
    _preparation.dispose();
    _practiceController?.dispose();
    super.dispose();
  }

  Future<void> _restore() async {
    final client = widget.practiceClient;
    if (client == null) {
      if (mounted) setState(() => _loading = false);
      return;
    }
    setState(() { _loading = true; _recoveryError = null; });
    try {
      final session = await client.getResumableSession();
      if (!mounted) return;
      if (session?.status == 'ACTIVE') {
        _practiceController = PracticeController(client: client, sessionID: session!.id, initialSession: session);
      } else if (session?.status == 'DRAFT') {
        _draftSession = session;
      }
    } on Exception catch (error) {
      if (mounted) _recoveryError = error;
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) return const Scaffold(body: Center(child: CircularProgressIndicator()));
    final controller = _practiceController;
    if (controller != null) return PracticePage(controller: controller);
    final draft = _draftSession;
    if (draft != null) return _ResumeDraftPage(session: draft, busy: _loading, onResume: _resumeDraft);
    if (_recoveryError != null) return _RecoveryFailure(onRetry: _restore, onContinue: () => setState(() => _recoveryError = null));
    return PreparationPage(controller: _preparation, onStartSession: _startSession);
  }

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

  Future<void> _resumeDraft() async {
    final client = widget.practiceClient;
    final draft = _draftSession;
    if (client == null || draft == null) return;
    setState(() => _loading = true);
    try {
      final active = await client.activateSession(draft.id);
      if (!mounted) return;
      setState(() { _draftSession = null; _practiceController = PracticeController(client: client, sessionID: active.id, initialSession: active); });
    } on Exception catch (error) {
      if (mounted) setState(() => _recoveryError = error);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }
}

final class _ResumeDraftPage extends StatelessWidget {
  const _ResumeDraftPage({required this.session, required this.busy, required this.onResume});
  final PracticeSession session;
  final bool busy;
  final Future<void> Function() onResume;

  @override
  Widget build(BuildContext context) => Scaffold(body: Center(child: Padding(padding: const EdgeInsets.all(24), child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.stretch, children: [
    const Text('发现一个未开始的练习', textAlign: TextAlign.center),
    const SizedBox(height: 12),
    FilledButton(onPressed: busy ? null : onResume, child: const Text('继续这次练习')),
  ]))));
}

final class _RecoveryFailure extends StatelessWidget {
  const _RecoveryFailure({required this.onRetry, required this.onContinue});
  final Future<void> Function() onRetry;
  final VoidCallback onContinue;

  @override
  Widget build(BuildContext context) => Scaffold(body: Center(child: Padding(padding: const EdgeInsets.all(24), child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.stretch, children: [
    const Text('未能恢复上次练习，请检查网络后重试。', textAlign: TextAlign.center),
    const SizedBox(height: 12),
    FilledButton(onPressed: onRetry, child: const Text('重试')),
    TextButton(onPressed: onContinue, child: const Text('进入场景选择')),
  ]))));
}
