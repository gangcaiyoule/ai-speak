import 'package:flutter/foundation.dart';

import '../scene/scene.dart';
import '../scene/scene_client.dart';
import '../practice_plan/practice_plan.dart';
import '../practice_plan/practice_plan_client.dart';

enum PreparationViewState { initial, loading, loaded, empty, failed }

final class PreparationController extends ChangeNotifier {
  PreparationController({required this.client, this.planClient});

  final SceneClient client;
  final PracticePlanClient? planClient;
  List<SceneDefinition> _scenes = const [];
  SceneDefinition? _selectedScene;
  SceneDefinition? _detail;
  RoleDefinition? _selectedRole;
  PracticeOption? _selectedOption;
  PreparationViewState _state = PreparationViewState.initial;
  String? _errorMessage;
  bool _loadingDetail = false;
  Future<void>? _sceneRequest;
  String? _failedSceneId;
  String _objective = '';
  bool _creatingPlan = false;
  PracticePlan? _createdPlan;
  String? _planErrorMessage;

  List<SceneDefinition> get scenes => List.unmodifiable(_scenes);
  SceneDefinition? get selectedScene => _selectedScene;
  SceneDefinition? get detail => _detail;
  List<RoleDefinition> get roles => _detail?.roles ?? const [];
  RoleDefinition? get selectedRole => _selectedRole;
  PracticeOption? get selectedOption => _selectedOption;
  PreparationViewState get state => _state;
  String? get errorMessage => _errorMessage;
  bool get isLoadingScenes => _state == PreparationViewState.loading;
  bool get isLoadingDetail => _loadingDetail;
  String get objective => _objective;
  bool get isCreatingPlan => _creatingPlan;
  PracticePlan? get createdPlan => _createdPlan;
  String? get planErrorMessage => _planErrorMessage;

  List<PracticeOption> get availableOptions {
    final role = _selectedRole;
    final detail = _detail;
    if (role == null || detail == null) return const [];
    return List.unmodifiable(detail.practiceOptions.where(
      (option) => option.roleId == null || option.roleId == role.id,
    ));
  }

  bool get hasCompleteSelection =>
      _selectedScene != null && _detail != null && _selectedRole != null && _selectedOption != null;

  SceneSelectionSnapshot? get selectionResult => hasCompleteSelection
      ? SceneSelectionSnapshot(
          scene: _detail!,
          selectedRoleIds: [_selectedRole!.id],
          practiceOptionId: _selectedOption!.id,
        )
      : null;

  Future<void> loadIfNeeded() {
    final request = _sceneRequest;
    if (request != null) return request;
    final operation = _loadScenes();
    _sceneRequest = operation;
    return operation;
  }

  Future<void> retryLastFailure() {
    final sceneId = _failedSceneId;
    if (sceneId == null) {
      _sceneRequest = null;
      return loadIfNeeded();
    }
    final scene = _firstOrNull(_scenes.where((item) => item.id == sceneId));
    return scene == null ? Future.value() : selectScene(scene);
  }

  Future<void> _loadScenes() async {
    _state = PreparationViewState.loading;
    _errorMessage = null;
    notifyListeners();
    try {
      _scenes = List.unmodifiable(await client.listScenes());
      _state = _scenes.isEmpty ? PreparationViewState.empty : PreparationViewState.loaded;
    } on SceneClientException catch (error) {
      _errorMessage = _messageFor(error);
      _state = PreparationViewState.failed;
      _failedSceneId = null;
    } finally {
      notifyListeners();
    }
  }

  Future<void> selectScene(SceneDefinition scene) async {
    final canonical = _firstOrNull(
      _scenes.where((item) => item.id == scene.id && item.version == scene.version),
    );
    if (canonical == null || _loadingDetail) return;
    _selectedScene = canonical;
    _detail = null;
    _selectedRole = null;
    _selectedOption = null;
    _loadingDetail = true;
    _errorMessage = null;
    _failedSceneId = canonical.id;
    notifyListeners();
    try {
      final detail = await client.getScene(canonical.id);
      _validateDetail(canonical, detail);
      _detail = detail;
    } on SceneClientException catch (error) {
      _errorMessage = _messageFor(error);
    } finally {
      _loadingDetail = false;
      notifyListeners();
    }
  }

  void selectRole(RoleDefinition role) {
    final canonical = _firstOrNull(roles.where((item) => item.id == role.id));
    if (canonical == null) return;
    _selectedRole = canonical;
    if (!availableOptions.any((option) => option.id == _selectedOption?.id)) {
      _selectedOption = null;
    }
    notifyListeners();
  }

  void selectOption(PracticeOption option) {
    if (availableOptions.every((item) => item.id != option.id)) return;
    _selectedOption = option;
    notifyListeners();
  }

  void setObjective(String value) { _objective = value; if (_planErrorMessage != null) { _planErrorMessage = null; notifyListeners(); } }

  Future<PracticePlan?> createPlan() async {
    final selection = selectionResult;
    final client = planClient;
    if (selection == null || client == null || _objective.trim().isEmpty || _creatingPlan) return null;
    _creatingPlan = true; _planErrorMessage = null; notifyListeners();
    try { _createdPlan = await client.createPlan(selection: selection, objective: _objective.trim()); return _createdPlan; }
    on PracticePlanClientException catch (error) { _planErrorMessage = error.message; return null; }
    finally { _creatingPlan = false; notifyListeners(); }
  }

  void clearSelection() {
    _selectedScene = null;
    _detail = null;
    _selectedRole = null;
    _selectedOption = null;
    _objective = '';
    _createdPlan = null;
    _planErrorMessage = null;
    _errorMessage = null;
    _failedSceneId = null;
    notifyListeners();
  }
}

void _validateDetail(SceneDefinition summary, SceneDefinition detail) {
  if (summary.id != detail.id || summary.version != detail.version || detail.roles.isEmpty || detail.practiceOptions.isEmpty) {
    throw const SceneClientException(kind: SceneClientFailureKind.invalidResponse);
  }
}

String _messageFor(SceneClientException error) => switch (error.kind) {
      SceneClientFailureKind.network || SceneClientFailureKind.unavailable => '场景暂时无法加载，请检查网络后重试。',
      SceneClientFailureKind.invalidResponse => '场景数据格式无效，请稍后重试。',
    };

T? _firstOrNull<T>(Iterable<T> values) {
  for (final value in values) {
    return value;
  }
  return null;
}
