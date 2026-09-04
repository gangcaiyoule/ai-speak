import 'package:flutter/material.dart';
import 'identity_client.dart';

final class AuthGate extends StatefulWidget {
  const AuthGate({required this.identityClient,required this.authenticatedBuilder,super.key});
  final IdentityClient identityClient; final WidgetBuilder authenticatedBuilder;
  @override State<AuthGate> createState()=>_AuthGateState();
}
class _AuthGateState extends State<AuthGate>{User? _user;bool _loading=true;bool _register=false;String? _error;String? _registeredEmail;
  @override void initState(){super.initState();_restore();}
  Future<void> _restore() async {try{_user=await widget.identityClient.currentUser();}on Object{_user=null;}if(mounted)setState(()=>_loading=false);}
  @override Widget build(BuildContext context){if(_loading)return const Scaffold(body:Center(child:CircularProgressIndicator()));final u=_user;if(u!=null)return widget.authenticatedBuilder(context);return _AuthPage(register:_register,initialEmail:_registeredEmail,error:_error,onSwitch:(){setState((){_register=!_register;_error=null;});},onSubmit:_submit);}
  Future<void> _submit(String email,String password) async {try{if(_register){await widget.identityClient.register(email:email,password:password);if(mounted)setState((){_register=false;_registeredEmail=email;_error='注册成功，请登录。';});}else{final result=await widget.identityClient.login(email:email,password:password);if(mounted)setState((){_user=result.user;_error=null;});}}on IdentityClientException catch(e){if(mounted)setState(()=>_error=e.message);}}
}
final class _AuthPage extends StatefulWidget{const _AuthPage({required this.register,required this.onSwitch,required this.onSubmit,this.error,this.initialEmail});final bool register;final String? error,initialEmail;final VoidCallback onSwitch;final Future<void> Function(String,String) onSubmit;@override State<_AuthPage> createState()=>_AuthPageState();}
class _AuthPageState extends State<_AuthPage>{late final TextEditingController _email=TextEditingController(text:widget.initialEmail);final _password=TextEditingController();final _form=GlobalKey<FormState>();bool _submitting=false;bool _obscure=true;
  @override void didUpdateWidget(covariant _AuthPage old){super.didUpdateWidget(old);if(widget.initialEmail!=old.initialEmail&&widget.initialEmail!=null)_email.text=widget.initialEmail!;}
  @override void dispose(){_email.dispose();_password.dispose();super.dispose();}
  @override Widget build(BuildContext context)=>Scaffold(appBar:AppBar(title:Text(widget.register?'注册 SpeakUp':'登录 SpeakUp')),body:SafeArea(child:SingleChildScrollView(padding:const EdgeInsets.all(24),child:Form(key:_form,child:Column(crossAxisAlignment:CrossAxisAlignment.stretch,children:[TextFormField(controller:_email,keyboardType:TextInputType.emailAddress,autofillHints:const [AutofillHints.email],decoration:const InputDecoration(labelText:'邮箱'),validator:(v)=>v==null||!v.contains('@')?'请输入有效邮箱':null),TextFormField(controller:_password,obscureText:_obscure,decoration:InputDecoration(labelText:'密码',suffixIcon:IconButton(icon:Icon(_obscure?Icons.visibility:Icons.visibility_off),onPressed:()=>setState(()=>_obscure=!_obscure))),validator:(v)=>v==null||v.length<8?'密码至少 8 位':null,onFieldSubmitted:_submitting?null:(_)=>_submit()),if(widget.error!=null)Padding(padding:const EdgeInsets.only(top:12),child:Text(widget.error!,style:TextStyle(color:Theme.of(context).colorScheme.error))),const SizedBox(height:16),FilledButton(onPressed:_submitting?null:_submit,child:_submitting?const SizedBox(width:20,height:20,child:CircularProgressIndicator(strokeWidth:2)):Text(widget.register?'注册':'登录')),TextButton(onPressed:_submitting?null:widget.onSwitch,child:Text(widget.register?'已有账号，去登录':'没有账号，去注册'))])))));
  Future<void> _submit([String _=''])async{if(!_form.currentState!.validate())return;setState(()=>_submitting=true);try{await widget.onSubmit(_email.text.trim(),_password.text);}finally{if(mounted)setState(()=>_submitting=false);}}
}
