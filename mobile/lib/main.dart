import 'package:flutter/material.dart';

import 'app/speak_up_app.dart';
import 'features/coaching/scene/wire_scene_client.dart';
import 'features/coaching/practice_plan/wire_practice_plan_client.dart';
import 'features/coaching/practice/wire_practice_client.dart';
import 'identity/identity_client.dart';

void main() {
  const baseUri = String.fromEnvironment(
    'SCENE_API_BASE_URL',
    defaultValue: 'http://127.0.0.1:8080',
  );
  final identityClient = WireIdentityClient(baseUri: Uri.parse(baseUri), store: const SecureSessionStore());
  runApp(SpeakUpApp(sceneClient: WireSceneClient(baseUri: Uri.parse(baseUri)), identityClient: identityClient, planClient: WirePracticePlanClient(baseUri: Uri.parse(baseUri), tokenProvider: identityClient.sessionToken), practiceClient: WirePracticeClient(baseUri: Uri.parse(baseUri), tokenProvider: identityClient.sessionToken)));
}
