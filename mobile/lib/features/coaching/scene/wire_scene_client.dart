import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

import 'scene.dart';
import 'scene_client.dart';
import 'scene_wire_codec.dart';

final class SceneWireResponse {
  const SceneWireResponse(this.statusCode, this.body);

  final int statusCode;
  final String body;
}

abstract interface class SceneWireTransport {
  Future<SceneWireResponse> get(Uri uri);
}

final class WireSceneClient implements SceneClient {
  WireSceneClient({
    required Uri baseUri,
    SceneWireTransport? transport,
    Duration requestTimeout = const Duration(seconds: 15),
  })  : _baseUri = baseUri,
        _transport = transport ?? _HttpSceneWireTransport(),
        _requestTimeout = requestTimeout {
    if (requestTimeout <= Duration.zero) {
      throw ArgumentError.value(requestTimeout, 'requestTimeout');
    }
  }

  final Uri _baseUri;
  final SceneWireTransport _transport;
  final Duration _requestTimeout;

  @override
  Future<List<SceneDefinition>> listScenes() async {
    final root = await _getJson('/v1/scenes');
    final scenes = root['scenes'];
    if (scenes is! List) throw _invalidResponse();
    final ids = <String>{};
    final result = scenes.map((value) {
      final scene = _decodeScene(value);
      if (scene.status != SceneStatus.active || !ids.add(scene.id)) {
        throw _invalidResponse();
      }
      return scene;
    }).toList(growable: false);
    return result;
  }

  @override
  Future<SceneDefinition> getScene(String sceneId) async {
    if (sceneId.trim().isEmpty || sceneId.contains('/') || sceneId.contains('\u0000')) {
      throw ArgumentError.value(sceneId, 'sceneId');
    }
    final scene = _decodeScene(
      await _getJson('/v1/scenes/${Uri.encodeComponent(sceneId)}'),
    );
    if (scene.id != sceneId || scene.status != SceneStatus.active) {
      throw _invalidResponse();
    }
    return scene;
  }

  Future<Map<String, Object?>> _getJson(String path) async {
    late final SceneWireResponse response;
    try {
      response = await _transport
          .get(_baseUri.resolve(path))
          .timeout(_requestTimeout);
    } on TimeoutException catch (error) {
      throw SceneClientException(kind: SceneClientFailureKind.network, message: error.toString());
    } on Exception catch (error) {
      // 网络层异常（原生平台为 SocketException/IOException，web 平台为
      // http.ClientException），统一归为 network 类失败。
      throw SceneClientException(kind: SceneClientFailureKind.network, message: error.toString());
    }
    if (response.statusCode == 404 || response.statusCode >= 500) {
      throw SceneClientException(
        kind: SceneClientFailureKind.unavailable,
        statusCode: response.statusCode,
      );
    }
    if (response.statusCode != 200) throw _invalidResponse();
    try {
      final decoded = jsonDecode(response.body);
      if (decoded is! Map) throw const SceneWireFormatException();
      return Map<String, Object?>.from(decoded);
    } on FormatException catch (error) {
      throw SceneClientException(kind: SceneClientFailureKind.invalidResponse, message: error.toString());
    } on SceneWireFormatException catch (error) {
      throw SceneClientException(kind: SceneClientFailureKind.invalidResponse, message: error.toString());
    }
  }
}

SceneDefinition _decodeScene(Object? value) {
  try {
    return decodeSceneDefinition(value);
  } on SceneWireFormatException catch (error) {
    throw SceneClientException(
      kind: SceneClientFailureKind.invalidResponse,
      message: error.toString(),
    );
  }
}

SceneClientException _invalidResponse() =>
    const SceneClientException(kind: SceneClientFailureKind.invalidResponse);

final class _HttpSceneWireTransport implements SceneWireTransport {
  final http.Client _client = http.Client();

  @override
  Future<SceneWireResponse> get(Uri uri) async {
    final response = await _client.get(uri);
    return SceneWireResponse(response.statusCode, response.body);
  }
}
