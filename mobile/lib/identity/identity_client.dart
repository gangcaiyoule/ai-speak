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
