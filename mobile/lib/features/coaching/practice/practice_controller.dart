import 'package:flutter/foundation.dart';

import '../coaching_clients.dart';
import 'wire_practice_client.dart';

enum PracticeViewState { initial, loading, ready, submitting, completed, failed }

final class PracticeController extends ChangeNotifier {
  PracticeController({required this.client, required this.sessionID, PracticeSession? initialSession}) : _session = initialSession;
  final PracticeClient client;
  final String sessionID;
  PracticeSession? _session;
  PracticeViewState _state = PracticeViewState.initial;
  String? _errorMessage;

  PracticeSession? get session => _session;
  PracticeViewState get state => _state;
  String? get errorMessage => _errorMessage;
  PracticeQuestion? get currentQuestion => _session?.currentQuestion;
  bool get isBusy => _state == PracticeViewState.loading || _state == PracticeViewState.submitting;

  Future<void> load() async {
    if (isBusy) return;
    _state = PracticeViewState.loading;
    _errorMessage = null;
    notifyListeners();
    try {
      _session = await client.getSession(sessionID);
      _state = _session?.status == 'COMPLETED' ? PracticeViewState.completed : PracticeViewState.ready;
    } on PracticeClientException catch (error) {
      _errorMessage = error.message;
      _state = PracticeViewState.failed;
    } on Exception catch (error) {
      _errorMessage = error.toString();
      _state = PracticeViewState.failed;
    } finally {
      notifyListeners();
    }
  }

  Future<bool> submit(String value) async {
    final question = currentQuestion;
    if (question == null || isBusy || _session?.status != 'ACTIVE') return false;
    final content = value.trim();
    if (content.isEmpty) {
      _errorMessage = '回答不能为空。';
      notifyListeners();
      return false;
    }
    _state = PracticeViewState.submitting;
    _errorMessage = null;
    notifyListeners();
    try {
      _session = await client.submitTextAnswer(sessionID, question.id, content);
      _state = _session?.status == 'COMPLETED' ? PracticeViewState.completed : PracticeViewState.ready;
      return true;
    } on PracticeClientException catch (error) {
      _errorMessage = error.message;
      _state = PracticeViewState.ready;
      return false;
    } on Exception catch (error) {
      _errorMessage = error.toString();
      _state = PracticeViewState.ready;
      return false;
    } finally {
      notifyListeners();
    }
  }

  Future<bool> complete() async {
    if (isBusy || _session?.status != 'ACTIVE' || currentQuestion != null) return false;
    _state = PracticeViewState.submitting;
    _errorMessage = null;
    notifyListeners();
    try {
      _session = await client.completeSession(sessionID);
      _state = PracticeViewState.completed;
      return true;
    } on PracticeClientException catch (error) {
      _errorMessage = error.message;
      _state = PracticeViewState.ready;
      return false;
    } on Exception catch (error) {
      _errorMessage = error.toString();
      _state = PracticeViewState.ready;
      return false;
    } finally {
      notifyListeners();
    }
  }

  Future<void> retry() => load();
}
