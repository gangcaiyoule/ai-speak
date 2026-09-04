import 'practice_plan.dart';
import '../scene/scene.dart';

abstract interface class PracticePlanClient {
  Future<PracticePlan> createPlan({required SceneSelectionSnapshot selection, required String objective});
  Future<List<PracticePlan>> listPlans();
  Future<PracticePlan> getPlan(String id);
  Future<PracticePlan> archivePlan(String id);
}
