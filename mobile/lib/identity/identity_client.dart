import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:http/http.dart' as http;

class User { const User({required this.id, required this.email}); final String id; final String email; }
class LoginResult { const LoginResult({required this.user, required this.token, required this.expiresAt}); final User user; final String token; final DateTime expiresAt; }
abstract interface class IdentityClient { Future<User> register({required String email, required String password}); Future<LoginResult> login({required String email, required String password}); Future<void> logout(); Future<User> currentUser(); }
abstract interface class SessionStore { Future<String?> readToken(); Future<void> writeToken(String token); Future<void> deleteToken(); }
final class SecureSessionStore implements SessionStore { const SecureSessionStore([this._storage = const FlutterSecureStorage()]); final FlutterSecureStorage _storage; static const _key='speakup.session.token'; @override Future<String?> readToken()=>_storage.read(key:_key); @override Future<void> writeToken(String token)=>_storage.write(key:_key,value:token); @override Future<void> deleteToken()=>_storage.delete(key:_key); }

final class WireIdentityClient implements IdentityClient {
  WireIdentityClient({required Uri baseUri, required SessionStore store}) : _baseUri=baseUri, _store=store;
  final Uri _baseUri; final SessionStore _store; final http.Client _http = http.Client(); String? _token;
  @override Future<User> register({required String email,required String password}) async { final r=await _request('POST','/v1/auth/register',body:{'email':email,'password':password}); if(r.statusCode!=201) throw _error(r); return _decodeUser(jsonDecode(r.body)['user']); }
  @override Future<LoginResult> login({required String email,required String password}) async { final r=await _request('POST','/v1/auth/login',body:{'email':email,'password':password}); if(r.statusCode!=200) throw _error(r); final result=_decodeLogin(jsonDecode(r.body)); await _store.writeToken(result.token); _token=result.token; return result; }
  @override Future<User> currentUser() async { final token=await _activeToken(); if(token==null) throw const IdentityClientException('authentication required',401); final r=await _request('GET','/v1/me',token:token); if(r.statusCode==401){await _clearToken();throw _error(r);} if(r.statusCode!=200) throw _error(r); return _decodeUser(jsonDecode(r.body)['user']); }
  Future<String?> sessionToken() => _activeToken();
  @override Future<void> logout() async { final token=await _activeToken(); try { if(token!=null){ final r=await _request('POST','/v1/auth/logout',token:token); if(r.statusCode!=204&&r.statusCode!=401) throw _error(r); } } finally { await _clearToken(); } }
  Future<String?> _activeToken() async=>_token??=await _store.readToken();
  Future<void> _clearToken() async{_token=null;await _store.deleteToken();}
  Future<_Response> _request(String method,String path,{Map<String,Object?>? body,String? token}) async { final request=http.Request(method,_baseUri.resolve(path))..headers['content-type']='application/json'; if(token!=null)request.headers['authorization']='Bearer $token'; if(body!=null)request.body=jsonEncode(body); final response=await _http.send(request); return _Response(response.statusCode,await response.stream.bytesToString()); }
  IdentityClientException _error(_Response r){String message='identity request failed';try{final v=jsonDecode(r.body);if(v is Map&&v['error'] is Map&&v['error']['message'] is String)message=v['error']['message'] as String;}catch(_){ }return IdentityClientException(message,r.statusCode);}
}
final class _Response{const _Response(this.statusCode,this.body);final int statusCode;final String body;}
LoginResult _decodeLogin(Object? value){if(value is! Map)throw const IdentityClientException('invalid login response');final user=_decodeUser(value['user']);final token=value['session_token'];final exp=value['expires_at'];if(token is! String||!token.startsWith('sess_')||exp is! String)throw const IdentityClientException('invalid login response');return LoginResult(user:user,token:token,expiresAt:DateTime.parse(exp));}
User _decodeUser(Object? value){if(value is! Map||value['id'] is! String||value['email'] is! String)throw const IdentityClientException('invalid user response');return User(id:value['id'] as String,email:value['email'] as String);}
final class IdentityClientException implements Exception{const IdentityClientException(this.message,[this.statusCode]);final String message;final int? statusCode;@override String toString()=>message;}
