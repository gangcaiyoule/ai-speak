import 'practice_plan.dart';
import '../scene/scene.dart';

/// @brief 练习计划客户端异常（HTTP 失败、响应非法等）。
final class PracticePlanClientException implements Exception {
  const PracticePlanClientException(this.message, [this.statusCode]);
  final String message;
  final int? statusCode;
  @override String toString() => message;
}

abstract interface class PracticePlanClient {
  Future<PracticePlan> createPlan({required SceneSelectionSnapshot selection, required String objective});
  Future<List<PracticePlan>> listPlans();
  Future<PracticePlan> getPlan(String id);
  Future<PracticePlan> archivePlan(String id);
}
