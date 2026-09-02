import 'evaluation_report.dart';

/// 表示一个口语练习场景。
class Scene { /// 创建场景值对象。
  const Scene({required this.id, required this.name});
  /// 服务端分配的场景标识。
  final String id;
  /// 面向用户展示的场景名称。
  final String name;
}
/// 表示一次口语练习会话。
class PracticeSession { /// 创建练习会话值对象。
  const PracticeSession({required this.id, required this.status});
  /// 服务端分配的会话标识。
  final String id;
  /// 当前练习会话状态。
  final String status;
}
/// 表示一次口语练习的评测报告。
class EvaluationReport { /// 创建评测报告摘要值对象。
  const EvaluationReport({
    required this.id,
    required this.summary,
    this.status = EvaluationStatus.queued,
    this.detail,
  });
  /// 服务端分配的报告标识。
  final String id;
  /// 面向用户展示的评测摘要。
  final String summary;
  /// 服务端评测任务状态；仅 ready 时 detail 有值。
  final EvaluationStatus status;
  /// 结构化报告详情，供后续复盘页面展示。
  final EvaluationReportDetail? detail;
}
/// 定义客户端读取练习场景的操作。
abstract interface class SceneClient { /// 读取全部可用练习场景。
  Future<List<Scene>> listScenes(); }
/// 定义客户端管理练习会话的操作。
abstract interface class PracticeClient {
  /// 创建指定场景的练习会话。
  Future<PracticeSession> createSession(String sceneID);
  /// 读取指定练习会话。
  Future<PracticeSession> getSession(String sessionID);
  /// 向指定练习会话提交文字回答。
  Future<void> submitTextAnswer(String sessionID, String questionID, String content);
  /// 完成指定练习会话。
  Future<void> completeSession(String sessionID);
}
/// 定义客户端读取练习评测的操作。
abstract interface class EvaluationClient {
  /// 读取指定练习会话的评测报告。
  Future<EvaluationReport> getBySession(String sessionID);
  /// 按报告标识读取评测报告。
  Future<EvaluationReport> getByID(String reportID);
}
