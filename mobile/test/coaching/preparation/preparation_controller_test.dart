import 'package:flutter_test/flutter_test.dart';
import 'package:ai_speak/features/coaching/preparation/preparation_controller.dart';
import 'package:ai_speak/features/coaching/scene/scene.dart';
import 'package:ai_speak/features/coaching/scene/scene_client.dart';

void main() {
  test('loads scenes in the client-provided order', () async {
    final first = _scene('first');
    final second = _scene('second');
    final controller = PreparationController(client: _FakeClient([first, second]));

    await controller.loadIfNeeded();

    expect(controller.state, PreparationViewState.loaded);
    expect(controller.scenes.map((scene) => scene.id), ['first', 'second']);
  });

  test('exposes an empty state without creating default scenes', () async {
    final controller = PreparationController(client: _FakeClient(const []));

    await controller.loadIfNeeded();

    expect(controller.state, PreparationViewState.empty);
    expect(controller.scenes, isEmpty);
  });

  test('shows failure and retries the list request', () async {
    final client = _FakeClient(const [], failuresBeforeSuccess: 1);
    final controller = PreparationController(client: client);

    await controller.loadIfNeeded();
    expect(controller.state, PreparationViewState.failed);
    await controller.retryLastFailure();

    expect(controller.state, PreparationViewState.loaded);
    expect(client.listCalls, 2);
  });

  test('loads detail and rejects a scene version mismatch', () async {
    final summary = _scene('scene', version: 2);
    final mismatched = _scene('scene', version: 1);
    final controller = PreparationController(
      client: _FakeClient([summary], details: {'scene': mismatched}),
    );
    await controller.loadIfNeeded();
    await controller.selectScene(summary);

    expect(controller.detail, isNull);
    expect(controller.errorMessage, isNotNull);
    expect(controller.hasCompleteSelection, isFalse);
  });

  test('filters practice options after switching roles', () async {
    final scene = _scene('scene', roles: 2);
    final controller = PreparationController(client: _FakeClient([scene], details: {'scene': scene}));
    await controller.loadIfNeeded();
    await controller.selectScene(scene);
    controller.selectRole(controller.roles.first);
    expect(controller.availableOptions.map((option) => option.id), ['full', 'first-focus']);
    controller.selectOption(controller.availableOptions.last);
    controller.selectRole(controller.roles.last);

    expect(controller.availableOptions.map((option) => option.id), ['full', 'second-focus']);
    expect(controller.selectedOption, isNull);
  });

  test('cannot start until role and option are selected', () async {
    final scene = _scene('scene');
    final controller = PreparationController(client: _FakeClient([scene], details: {'scene': scene}));
    await controller.loadIfNeeded();
    await controller.selectScene(scene);

    expect(controller.hasCompleteSelection, isFalse);
    controller.selectRole(controller.roles.first);
    expect(controller.hasCompleteSelection, isFalse);
  });

  test('produces the complete scene selection result', () async {
    final scene = _scene('scene');
    final controller = PreparationController(client: _FakeClient([scene], details: {'scene': scene}));
    await controller.loadIfNeeded();
    await controller.selectScene(scene);
    controller.selectRole(controller.roles.first);
    controller.selectOption(controller.availableOptions.first);

    final result = controller.selectionResult!;
    expect(result.scene.id, 'scene');
    expect(result.scene.version, 1);
    expect(result.selectedRoleIds, ['role-1']);
    expect(result.practiceOptionId, 'full');
  });
}

final class _FakeClient implements SceneClient {
  _FakeClient(this.scenes, {this.details = const {}, this.failuresBeforeSuccess = 0});

  final List<SceneDefinition> scenes;
  final Map<String, SceneDefinition> details;
  int failuresBeforeSuccess;
  int listCalls = 0;

  @override
  Future<List<SceneDefinition>> listScenes() async {
    listCalls++;
    if (failuresBeforeSuccess > 0) {
      failuresBeforeSuccess--;
      throw const SceneClientException(kind: SceneClientFailureKind.network);
    }
    return scenes;
  }

  @override
  Future<SceneDefinition> getScene(String sceneId) async => details[sceneId] ?? scenes.firstWhere((scene) => scene.id == sceneId);
}

SceneDefinition _scene(String id, {int version = 1, int roles = 1}) {
  final roleDefinitions = List< RoleDefinition>.generate(roles, (index) => RoleDefinition(
    id: 'role-${index + 1}',
    sceneId: id,
    type: 'INTERVIEWER',
    displayName: 'Role ${index + 1}',
    responsibilities: 'Ask useful questions.',
    style: 'Structured.',
    practiceObjectives: const [RolePracticeObjective(objectiveId: 'clarity', description: 'Speak clearly.')],
  ));
  final options = <PracticeOption>[
    PracticeOption(id: 'full', sceneId: id, mode: PracticeMode.fullSimulation, displayName: 'Full', suggestedDurationSeconds: 600, roleId: null),
    for (var index = 0; index < roles; index++)
      PracticeOption(id: '${index == 0 ? 'first' : 'second'}-focus', sceneId: id, mode: PracticeMode.focus, displayName: 'Focus', suggestedDurationSeconds: 300, roleId: 'role-${index + 1}'),
  ];
  return SceneDefinition(
    id: id,
    experience: PracticeExperience.interview,
    category: SceneCategory.interviewRecruiter,
    name: id,
    version: version,
    status: SceneStatus.active,
    prompt: const ScenePrompt(publicSceneBrief: 'Brief.', practiceGoal: 'Goal.', userRole: 'Candidate', aiRole: 'Interviewer', personaSummary: 'Structured.', focusAreas: ['clarity'], turnBlueprints: []),
    roles: roleDefinitions,
    practiceOptions: options,
  );
}
