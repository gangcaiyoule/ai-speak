import 'package:flutter/material.dart';

import 'identity_client.dart';

/// 在恢复会话、登录注册和练习页面之间切换。
final class AuthGate extends StatefulWidget {
  const AuthGate({required this.identityClient, required this.authenticatedBuilder, super.key});

  final IdentityClient identityClient;
  final WidgetBuilder authenticatedBuilder;

  @override
  State<AuthGate> createState() => _AuthGateState();
}

class _AuthGateState extends State<AuthGate> {
  User? _user;
  bool _loading = true;
  bool _register = false;
  String? _error;

  @override
  void initState() { super.initState(); _restore(); }

  Future<void> _restore() async {
    try { _user = await widget.identityClient.currentUser(); } on Object { _user = null; }
    if (mounted) setState(() => _loading = false);
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) return const Scaffold(body: Center(child: CircularProgressIndicator()));
    final user = _user;
    return user == null
        ? _AuthPage(register: _register, error: _error, onSwitch: () => setState(() { _register = !_register; _error = null; }), onSubmit: _submit)
        : widget.authenticatedBuilder(context);
  }

  Future<void> _submit(String email, String password) async {
    try {
      final session = _register ? await widget.identityClient.register(email: email, password: password) : await widget.identityClient.login(email: email, password: password);
      if (mounted) setState(() { _user = session.user; _error = null; });
    } on IdentityClientException catch (error) { if (mounted) setState(() => _error = error.message); }
  }
}

final class _AuthPage extends StatefulWidget {
  const _AuthPage({required this.register, required this.onSwitch, required this.onSubmit, this.error});
  final bool register;
  final String? error;
  final VoidCallback onSwitch;
  final Future<void> Function(String email, String password) onSubmit;
  @override
  State<_AuthPage> createState() => _AuthPageState();
}

class _AuthPageState extends State<_AuthPage> {
  final _email = TextEditingController();
  final _password = TextEditingController();
  bool _submitting = false;
  @override
  void dispose() { _email.dispose(); _password.dispose(); super.dispose(); }
  @override
  Widget build(BuildContext context) => Scaffold(
        appBar: AppBar(title: Text(widget.register ? '注册 SpeakUp' : '登录 SpeakUp')),
        body: Padding(padding: const EdgeInsets.all(24), child: Column(children: [
          TextField(controller: _email, keyboardType: TextInputType.emailAddress, decoration: const InputDecoration(labelText: '邮箱')),
          TextField(controller: _password, obscureText: true, decoration: const InputDecoration(labelText: '密码')),
          if (widget.error != null) Text(widget.error!, style: TextStyle(color: Theme.of(context).colorScheme.error)),
          const SizedBox(height: 16),
          FilledButton(onPressed: _submitting ? null : _submit, child: Text(widget.register ? '注册' : '登录')),
          TextButton(onPressed: _submitting ? null : widget.onSwitch, child: Text(widget.register ? '已有账号，去登录' : '没有账号，去注册')),
        ])),
      );
  Future<void> _submit() async { if (_email.text.trim().isEmpty || _password.text.isEmpty) return; setState(() => _submitting = true); try { await widget.onSubmit(_email.text.trim(), _password.text); } finally { if (mounted) setState(() => _submitting = false); } }
}
