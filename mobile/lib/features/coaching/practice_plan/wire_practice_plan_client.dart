import 'dart:convert';
import 'dart:io';

import '../scene/scene.dart';
import 'practice_plan.dart';
import 'practice_plan_client.dart';

final class PracticePlanClientException implements Exception {
  const PracticePlanClientException(this.message, [this.statusCode]);
  final String message;
  final int? statusCode;
  @override String toString() => message;
}

final class WirePracticePlanClient implements PracticePlanClient {
  WirePracticePlanClient({required Uri baseUri, required Future<String?> Function() tokenProvider}) : _baseUri = baseUri, _tokenProvider = tokenProvider;
  final Uri _baseUri;
  final Future<String?> Function() _tokenProvider;
  final HttpClient _http = HttpClient();

  @override Future<PracticePlan> createPlan({required SceneSelectionSnapshot selection, required String objective}) async {
    final response = await _request('POST', '/v1/practice-plans', body: {'scene_id': selection.scene.id, 'scene_version': selection.scene.version, 'role_id': selection.selectedRoleIds.single, 'practice_option_id': selection.practiceOptionId, 'objective': objective});
    if (response.statusCode != HttpStatus.created) throw _error(response);
    return _decode(jsonDecode(response.body) is Map ? (jsonDecode(response.body) as Map)['plan'] : null);
  }
  @override Future<List<PracticePlan>> listPlans() async { final response = await _request('GET', '/v1/practice-plans'); if (response.statusCode != HttpStatus.ok) throw _error(response); final root = jsonDecode(response.body); if (root is! Map || root['plans'] is! List) throw const PracticePlanClientException('invalid practice plan response'); return (root['plans'] as List).map(_decode).toList(growable: false); }
  @override Future<PracticePlan> getPlan(String id) async { final response = await _request('GET', '/v1/practice-plans/${Uri.encodeComponent(id)}'); if (response.statusCode != HttpStatus.ok) throw _error(response); final root = jsonDecode(response.body); return _decode(root is Map ? root['plan'] : null); }
  @override Future<PracticePlan> archivePlan(String id) async { final response = await _request('POST', '/v1/practice-plans/${Uri.encodeComponent(id)}/archive'); if (response.statusCode != HttpStatus.ok) throw _error(response); final root = jsonDecode(response.body); return _decode(root is Map ? root['plan'] : null); }
  Future<_Response> _request(String method, String path, {Map<String, Object?>? body}) async { final token = await _tokenProvider(); if (token == null || token.isEmpty) throw const PracticePlanClientException('authentication required', HttpStatus.unauthorized); final request = await _http.openUrl(method, _baseUri.resolve(path)); request.headers.set(HttpHeaders.authorizationHeader, 'Bearer $token'); request.headers.contentType = ContentType.json; if (body != null) request.write(jsonEncode(body)); final response = await request.close(); return _Response(response.statusCode, await utf8.decoder.bind(response).join()); }
  PracticePlanClientException _error(_Response response) { var message = 'practice plan request failed'; try { final root = jsonDecode(response.body); if (root is Map && root['error'] is Map && root['error']['message'] is String) message = root['error']['message'] as String; } catch (_) {} return PracticePlanClientException(message, response.statusCode); }
}
PracticePlan _decode(Object? value) { if (value is! Map || value['id'] is! String || value['scene_id'] is! String || value['scene_version'] is! int || value['role_id'] is! String || value['practice_option_id'] is! String || value['objective'] is! String || value['status'] is! String) throw const PracticePlanClientException('invalid practice plan response'); return PracticePlan(id: value['id'] as String, sceneId: value['scene_id'] as String, sceneVersion: value['scene_version'] as int, roleId: value['role_id'] as String, practiceOptionId: value['practice_option_id'] as String, objective: value['objective'] as String, status: value['status'] as String); }
final class _Response { const _Response(this.statusCode, this.body); final int statusCode; final String body; }
