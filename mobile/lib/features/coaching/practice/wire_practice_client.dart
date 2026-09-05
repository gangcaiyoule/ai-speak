import 'dart:convert';
import 'dart:io';

import '../coaching_clients.dart';

final class PracticeClientException implements Exception {
  const PracticeClientException(this.message, [this.statusCode]);
  final String message;
  final int? statusCode;
  @override
  String toString() => message;
}

final class WirePracticeClient implements PracticeClient {
  WirePracticeClient({required Uri baseUri, required Future<String?> Function() tokenProvider}) : _baseUri = baseUri, _tokenProvider = tokenProvider;
  final Uri _baseUri;
  final Future<String?> Function() _tokenProvider;
  final HttpClient _http = HttpClient();

  @override
  Future<PracticeSession> createSession(String planID) async {
    final response = await _request('POST', '/v1/practice-sessions', body: {'plan_id': planID});
    if (response.statusCode != HttpStatus.created) throw _error(response);
    return _decode(_map(response)['session']);
  }

  @override
  Future<PracticeSession> activateSession(String sessionID) async {
    final response = await _request('POST', '/v1/practice-sessions/${Uri.encodeComponent(sessionID)}/activation');
    if (response.statusCode != HttpStatus.ok) throw _error(response);
    return _decode(_map(response)['session']);
  }

  @override
  Future<PracticeSession> getSession(String sessionID) async {
    final response = await _request('GET', '/v1/practice-sessions/${Uri.encodeComponent(sessionID)}');
    if (response.statusCode != HttpStatus.ok) throw _error(response);
    return _decode(_map(response)['session']);
  }

  @override
  Future<PracticeSession> submitTextAnswer(String sessionID, String questionID, String content) async {
    final response = await _request('POST', '/v1/practice-sessions/${Uri.encodeComponent(sessionID)}/text-answers', body: {'question_id': questionID, 'content': content});
    if (response.statusCode != HttpStatus.ok) throw _error(response);
    return _decode(_map(response)['session']);
  }

  @override
  Future<PracticeSession> completeSession(String sessionID) async {
    final response = await _request('POST', '/v1/practice-sessions/${Uri.encodeComponent(sessionID)}/complete');
    if (response.statusCode != HttpStatus.ok) throw _error(response);
    return _decode(_map(response)['session']);
  }

  Future<_Response> _request(String method, String path, {Map<String, Object?>? body}) async {
    final token = await _tokenProvider();
    if (token == null || token.isEmpty) throw const PracticeClientException('authentication required', HttpStatus.unauthorized);
    try {
      final request = await _http.openUrl(method, _baseUri.resolve(path));
      request.headers.set(HttpHeaders.authorizationHeader, 'Bearer $token');
      request.headers.contentType = ContentType.json;
      if (body != null) request.write(jsonEncode(body));
      final response = await request.close();
      return _Response(response.statusCode, await utf8.decoder.bind(response).join());
    } on IOException catch (error) {
      throw PracticeClientException('练习服务暂时不可用：$error');
    }
  }

  PracticeClientException _error(_Response response) {
    var message = 'practice request failed';
    try {
      final root = jsonDecode(response.body);
      if (root is Map && root['error'] is Map && root['error']['message'] is String) message = root['error']['message'] as String;
    } catch (_) {}
    return PracticeClientException(message, response.statusCode);
  }
}

Map<String, dynamic> _map(_Response response) {
  final value = jsonDecode(response.body);
  if (value is! Map) throw const PracticeClientException('invalid practice response');
  return value.cast<String, dynamic>();
}

PracticeSession _decode(Object? value) {
  if (value is! Map || value['id'] is! String || value['status'] is! String) throw const PracticeClientException('invalid practice session response');
  final current = value['current_question'];
  return PracticeSession(
    id: value['id'] as String,
    status: value['status'] as String,
    planId: value['plan_id'] as String? ?? '',
    currentQuestionId: value['current_question_id'] as String?,
    currentQuestion: current is Map ? _question(current) : null,
  );
}

PracticeQuestion _question(Map value) {
  if (value['id'] is! String || value['session_id'] is! String || value['position'] is! int || value['content'] is! String) throw const PracticeClientException('invalid practice question response');
  return PracticeQuestion(id: value['id'] as String, sessionId: value['session_id'] as String, position: value['position'] as int, content: value['content'] as String);
}

final class _Response {
  const _Response(this.statusCode, this.body);
  final int statusCode;
  final String body;
}
