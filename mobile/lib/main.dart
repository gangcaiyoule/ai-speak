import 'package:flutter/material.dart';

import 'app/speak_up_app.dart';
import 'features/coaching/scene/wire_scene_client.dart';

void main() {
  const baseUri = String.fromEnvironment(
    'SCENE_API_BASE_URL',
    defaultValue: 'http://127.0.0.1:8080',
  );
  runApp(SpeakUpApp(sceneClient: WireSceneClient(baseUri: Uri.parse(baseUri))));
}
