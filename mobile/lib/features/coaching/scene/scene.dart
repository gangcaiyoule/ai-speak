/// 场景目录使用的领域模型。
enum PracticeExperience {
  interview('INTERVIEW', '英文面试'),
  ieltsSpeaking('IELTS_SPEAKING', 'IELTS 口语'),
  workplace('WORKPLACE', '职场英语'),
  lifeAndTravel('LIFE_AND_TRAVEL', '生活与旅行');

  const PracticeExperience(this.wireValue, this.displayName);

  final String wireValue;
  final String displayName;

  static PracticeExperience fromWireValue(String value) => values.firstWhere(
        (item) => item.wireValue == value,
        orElse: () => throw const SceneWireFormatException(),
      );
}

enum SceneCategory {
  interviewRecruiter('INTERVIEW_RECRUITER'),
  interviewBehavioral('INTERVIEW_BEHAVIORAL'),
  interviewProfessional('INTERVIEW_PROFESSIONAL'),
  interviewHiringManager('INTERVIEW_HIRING_MANAGER'),
  interviewCustom('INTERVIEW_CUSTOM'),
  ieltsSpeaking('IELTS_SPEAKING'),
  workplaceGeneral('WORKPLACE_GENERAL'),
  lifeTravel('LIFE_TRAVEL'),
  lifeDaily('LIFE_DAILY');

  const SceneCategory(this.wireValue);

  final String wireValue;

  static SceneCategory fromWireValue(String value) => values.firstWhere(
        (item) => item.wireValue == value,
        orElse: () => throw const SceneWireFormatException(),
      );
}

enum SceneStatus { active, inactive }

final class ScenePrompt {
  const ScenePrompt({
    required this.publicSceneBrief,
    required this.practiceGoal,
    required this.userRole,
    required this.aiRole,
    required this.personaSummary,
    required this.focusAreas,
    required this.turnBlueprints,
  });

  final String publicSceneBrief;
  final String practiceGoal;
  final String userRole;
  final String aiRole;
  final String personaSummary;
  final List<String> focusAreas;
  final List<String> turnBlueprints;
}

final class RolePracticeObjective {
  const RolePracticeObjective({required this.objectiveId, required this.description});

  final String objectiveId;
  final String description;
}

final class RoleDefinition {
  const RoleDefinition({
    required this.id,
    required this.sceneId,
    required this.type,
    required this.displayName,
    required this.responsibilities,
    required this.style,
    required this.practiceObjectives,
  });

  final String id;
  final String sceneId;
  final String type;
  final String displayName;
  final String responsibilities;
  final String style;
  final List<RolePracticeObjective> practiceObjectives;
}

enum PracticeMode {
  fullSimulation('FULL_SIMULATION', '完整模拟'),
  focus('FOCUS', '重点练习'),
  fullMock('FULL_MOCK', '完整模拟考试'),
  part1('PART_1', 'Part 1'),
  part2('PART_2', 'Part 2'),
  part3('PART_3', 'Part 3');

  const PracticeMode(this.wireValue, this.displayName);

  final String wireValue;
  final String displayName;

  static PracticeMode fromWireValue(String value) => values.firstWhere(
        (item) => item.wireValue == value,
        orElse: () => throw const SceneWireFormatException(),
      );
}

final class PracticeOption {
  const PracticeOption({
    required this.id,
    required this.sceneId,
    required this.mode,
    required this.displayName,
    required this.suggestedDurationSeconds,
    this.turnPolicyRef = '',
    this.sessionPolicyRef = '',
    this.evaluationPolicyRef = '',
    this.roleId,
  });

  final String id;
  final String sceneId;
  final PracticeMode mode;
  final String displayName;
  final int suggestedDurationSeconds;
  final String turnPolicyRef;
  final String sessionPolicyRef;
  final String evaluationPolicyRef;
  final String? roleId;
}

final class SceneDefinition {
  const SceneDefinition({
    required this.id,
    required this.experience,
    required this.category,
    required this.name,
    required this.version,
    required this.status,
    required this.prompt,
    required this.roles,
    required this.practiceOptions,
  });

  final String id;
  final PracticeExperience experience;
  final SceneCategory category;
  final String name;
  final int version;
  final SceneStatus status;
  final ScenePrompt prompt;
  final List<RoleDefinition> roles;
  final List<PracticeOption> practiceOptions;
}

final class SceneSelectionSnapshot {
  const SceneSelectionSnapshot({
    required this.scene,
    required this.selectedRoleIds,
    required this.practiceOptionId,
  });

  final SceneDefinition scene;
  final List<String> selectedRoleIds;
  final String practiceOptionId;
}

/// 网络响应无法解释时必须显式失败，不能回退成默认场景。
final class SceneWireFormatException implements Exception {
  const SceneWireFormatException([this.message = 'invalid scene response']);

  final String message;

  @override
  String toString() => 'SceneWireFormatException: $message';
}
