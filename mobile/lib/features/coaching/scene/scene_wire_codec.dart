import 'scene.dart';

SceneDefinition decodeSceneDefinition(Object? value) {
  final object = _object(value, const {
    'scene_id',
    'practice_experience',
    'scene_category',
    'name',
    'scene_version',
    'status',
    'prompt',
    'roles',
    'practice_options',
  });
  final roles = _list(object['roles']);
  final options = _list(object['practice_options']);
  if (roles.isEmpty || options.isEmpty) throw const SceneWireFormatException();

  final sceneId = _id(object['scene_id']);
  final roleIds = <String>{};
  final decodedRoles = roles.map((value) {
    final role = _decodeRole(value);
    if (role.sceneId != sceneId || !roleIds.add(role.id)) {
      throw const SceneWireFormatException();
    }
    return role;
  }).toList(growable: false);
  final optionIds = <String>{};
  final decodedOptions = options.map((value) {
    final option = _decodeOption(value);
    if (option.sceneId != sceneId ||
        !optionIds.add(option.id) ||
        (option.roleId != null && !roleIds.contains(option.roleId))) {
      throw const SceneWireFormatException();
    }
    return option;
  }).toList(growable: false);

  final prompt = _object(object['prompt'], const {
    'public_scene_brief',
    'practice_goal',
    'user_role',
    'ai_role',
    'persona_summary',
    'focus_areas',
  });
  final turnBlueprints = prompt.containsKey('turn_blueprints')
      ? _stringList(prompt['turn_blueprints'])
      : const <String>[];
  return SceneDefinition(
    id: sceneId,
    experience: PracticeExperience.fromWireValue(_string(object['practice_experience'])),
    category: SceneCategory.fromWireValue(_string(object['scene_category'])),
    name: _nonBlank(object['name']),
    version: _positiveInt(object['scene_version']),
    status: switch (_string(object['status'])) {
      'active' => SceneStatus.active,
      'inactive' => SceneStatus.inactive,
      _ => throw const SceneWireFormatException(),
    },
    prompt: ScenePrompt(
      publicSceneBrief: _nonBlank(prompt['public_scene_brief']),
      practiceGoal: _nonBlank(prompt['practice_goal']),
      userRole: _nonBlank(prompt['user_role']),
      aiRole: _nonBlank(prompt['ai_role']),
      personaSummary: _nonBlank(prompt['persona_summary']),
      focusAreas: _stringList(prompt['focus_areas']),
      turnBlueprints: turnBlueprints,
    ),
    roles: decodedRoles,
    practiceOptions: decodedOptions,
  );
}

Map<String, Object?> _object(Object? value, Set<String> required) {
  if (value is! Map) throw const SceneWireFormatException();
  final result = Map<String, Object?>.from(value);
  if (!result.keys.toSet().containsAll(required)) {
    throw const SceneWireFormatException();
  }
  return result;
}

List<Object?> _list(Object? value) {
  if (value is! List) throw const SceneWireFormatException();
  return List<Object?>.from(value);
}

RoleDefinition _decodeRole(Object? value) {
  final object = _object(value, {
    'role_definition_id',
    'scene_id',
    'role_type',
    'display_name',
    'responsibilities',
    'style',
    'practice_objectives',
  });
  final objectives = _list(object['practice_objectives']).map((value) {
    final item = _object(value, {'objective_id', 'description'});
    return RolePracticeObjective(
      objectiveId: _id(item['objective_id']),
      description: _nonBlank(item['description']),
    );
  }).toList(growable: false);
  if (objectives.isEmpty) throw const SceneWireFormatException();
  return RoleDefinition(
    id: _id(object['role_definition_id']),
    sceneId: _id(object['scene_id']),
    type: _nonBlank(object['role_type']),
    displayName: _nonBlank(object['display_name']),
    responsibilities: _nonBlank(object['responsibilities']),
    style: _nonBlank(object['style']),
    practiceObjectives: objectives,
  );
}

PracticeOption _decodeOption(Object? value) {
  final object = _object(value, {
    'practice_option_id',
    'scene_id',
    'practice_mode',
    'display_name',
    'suggested_duration_seconds',
    'turn_policy_ref',
    'session_policy_ref',
    'evaluation_policy_ref',
  });
  final roleId = object['role_definition_id'];
  if (roleId != null && roleId is! String) throw const SceneWireFormatException();
  return PracticeOption(
    id: _id(object['practice_option_id']),
    sceneId: _id(object['scene_id']),
    mode: PracticeMode.fromWireValue(_string(object['practice_mode'])),
    displayName: _nonBlank(object['display_name']),
    suggestedDurationSeconds: _positiveInt(object['suggested_duration_seconds']),
    turnPolicyRef: _nonBlank(object['turn_policy_ref']),
    sessionPolicyRef: _nonBlank(object['session_policy_ref']),
    evaluationPolicyRef: _nonBlank(object['evaluation_policy_ref']),
    roleId: roleId as String?,
  );
}

String _id(Object? value) {
  final result = _nonBlank(value);
  if (result.length > 128) throw const SceneWireFormatException();
  return result;
}

String _nonBlank(Object? value) {
  if (value is! String || value.trim().isEmpty || value != value.trim()) {
    throw const SceneWireFormatException();
  }
  return value;
}

String _string(Object? value) {
  if (value is! String) throw const SceneWireFormatException();
  return value;
}

int _positiveInt(Object? value) {
  if (value is! int || value < 1) throw const SceneWireFormatException();
  return value;
}

List<String> _stringList(Object? value) {
  final values = _list(value).map(_nonBlank).toList(growable: false);
  if (values.isEmpty || values.toSet().length != values.length) {
    throw const SceneWireFormatException();
  }
  return values;
}
