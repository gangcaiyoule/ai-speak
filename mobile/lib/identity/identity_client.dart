import 'dart:convert';
import 'dart:io';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// 表示身份接口返回的已认证用户。
class User {
  /// 创建一个用户值对象。
  const User({required this.id, required this.email});

  /// 服务端分配的用户标识。
  final String id;

  /// 用户邮箱地址。
  final String email;
}

/// 表示不透明的已认证会话。
class Session {
  /// 创建一个会话值对象。
  const Session({required this.id, required this.token, required this.user});

  /// 服务端分配的会话标识。
  final String id;

  /// 受保护请求使用的 Bearer 令牌。
  final String token;

  /// 此会话对应的用户。
  final User user;
}

/// 定义客户端管理账户和会话的操作。
abstract interface class IdentityClient {
  /// 注册新用户并创建会话。
  Future<Session> register({required String email, required String password});

  /// 验证已有用户并创建会话。
  Future<Session> login({required String email, required String password});

  /// 使当前会话失效。
  Future<void> logout();

  /// 加载当前会话关联的用户。
  Future<User> currentUser();
}

/// 持久化当前会话 Token 的抽象，便于测试时替换安全存储。
abstract interface class SessionStore {
  Future<String?> readToken();

  Future<void> writeToken(String token);

  Future<void> deleteToken();
}

/// 使用平台安全存储保存不透明会话 Token。
final class SecureSessionStore implements SessionStore {
  const SecureSessionStore([this._storage = const FlutterSecureStorage()]);

  final FlutterSecureStorage _storage;
  static const _key = 'speakup.session.token';

  @override
  Future<String?> readToken() => _storage.read(key: _key);

  @override
  Future<void> writeToken(String token) => _storage.write(key: _key, value: token);

  @override
  Future<void> deleteToken() => _storage.delete(key: _key);
}

/// HTTP 实现，负责认证请求和会话 Token 的本地保存。
final class WireIdentityClient implements IdentityClient {
  WireIdentityClient({required Uri baseUri, required SessionStore store}) : _baseUri = baseUri, _store = store;

  final Uri _baseUri;
  final SessionStore _store;
  final HttpClient _http = HttpClient();
  String? _token;

  @override
  Future<Session> register({required String email, required String password}) => _authenticate('/v1/auth/register', email, password);

  @override
  Future<Session> login({required String email, required String password}) => _authenticate('/v1/auth/login', email, password);

  @override
  Future<void> logout() async {
    final token = await _activeToken();
    if (token != null) {
      final request = await _http.postUrl(_baseUri.resolve('/v1/auth/logout'));
      request.headers.set(HttpHeaders.authorizationHeader, 'Bearer $token');
      final response = await request.close();
      await response.drain<void>();
      if (response.statusCode != HttpStatus.noContent && response.statusCode != HttpStatus.unauthorized) {
        throw IdentityClientException('logout failed', response.statusCode);
      }
    }
    _token = null;
    await _store.deleteToken();
  }

  @override
  Future<User> currentUser() async {
    final token = await _activeToken();
    if (token == null) throw const IdentityClientException('authentication required', HttpStatus.unauthorized);
    final response = await _request(HttpMethod.get, '/v1/me', token: token);
    if (response.statusCode != HttpStatus.ok) throw _error(response);
    return _decodeUser(jsonDecode(response.body));
  }

  Future<Session> _authenticate(String path, String email, String password) async {
    final response = await _request(HttpMethod.post, path, body: {'email': email, 'password': password});
    if (response.statusCode != HttpStatus.ok && response.statusCode != HttpStatus.created) throw _error(response);
    final session = _decodeSession(jsonDecode(response.body));
    _token = session.token;
    await _store.writeToken(session.token);
    return session;
  }

  Future<String?> _activeToken() async => _token ??= await _store.readToken();

  Future<_Response> _request(HttpMethod method, String path, {Map<String, String>? headers, Map<String, Object?>? body, String? token}) async {
    final request = await _http.openUrl(method == HttpMethod.get ? 'GET' : 'POST', _baseUri.resolve(path));
    request.headers.contentType = ContentType.json;
    if (token != null) request.headers.set(HttpHeaders.authorizationHeader, 'Bearer $token');
    headers?.forEach(request.headers.set);
    if (body != null) request.write(jsonEncode(body));
    final response = await request.close();
    return _Response(response.statusCode, await utf8.decoder.bind(response).join());
  }

  IdentityClientException _error(_Response response) => IdentityClientException('identity request failed', response.statusCode);
}

final class _Response {
  const _Response(this.statusCode, this.body);
  final int statusCode;
  final String body;
}

enum HttpMethod { get, post }

Session _decodeSession(Object? value) {
  if (value is! Map) throw const IdentityClientException('invalid session response');
  final user = _decodeUser(value['user']);
  final id = value['id'];
  final token = value['token'];
  if (id is! String || id.isEmpty || token is! String || !token.startsWith('sess_')) throw const IdentityClientException('invalid session response');
  return Session(id: id, token: token, user: user);
}

User _decodeUser(Object? value) {
  if (value is! Map || value['id'] is! String || value['email'] is! String) throw const IdentityClientException('invalid user response');
  return User(id: value['id'] as String, email: value['email'] as String);
}

final class IdentityClientException implements Exception {
  const IdentityClientException(this.message, [this.statusCode]);
  final String message;
  final int? statusCode;
  @override
  String toString() => message;
}
