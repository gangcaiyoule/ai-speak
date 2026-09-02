import 'scene.dart';

enum SceneClientFailureKind { network, unavailable, invalidResponse }

final class SceneClientException implements Exception {
  const SceneClientException({
    required this.kind,
    this.statusCode,
    this.message,
  });

  final SceneClientFailureKind kind;
  final int? statusCode;
  final String? message;

  @override
  String toString() => message ?? 'SceneClientException(${kind.name})';
}

abstract interface class SceneClient {
  Future<List<SceneDefinition>> listScenes();

  Future<SceneDefinition> getScene(String sceneId);
}
