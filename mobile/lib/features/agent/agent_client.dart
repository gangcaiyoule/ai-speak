/// 表示一条 AI 对话线程。
class AgentThread { /// 创建对话线程值对象。
  const AgentThread({required this.id});
  /// 服务端分配的线程标识。
  final String id;
}
/// 表示对话线程中的一条消息。
class AgentMessage { /// 创建对话消息值对象。
  const AgentMessage({required this.id, required this.role, required this.content});
  /// 服务端分配的消息标识。
  final String id;
  /// 消息发送方角色。
  final String role;
  /// 消息文本内容。
  final String content;
}
/// 表示一次 Agent 生成任务。
class AgentRun { /// 创建 Agent 任务值对象。
  const AgentRun({required this.id, required this.status});
  /// 服务端分配的任务标识。
  final String id;
  /// 当前任务状态。
  final String status;
}
/// 定义客户端访问 Agent 对话能力的操作。
abstract interface class AgentClient {
  /// 创建一条新的对话线程。
  Future<AgentThread> createThread();
  /// 读取当前用户的对话线程。
  Future<List<AgentThread>> listThreads();
  /// 读取指定对话线程。
  Future<AgentThread> getThread(String threadID);
  /// 向指定线程发送文本消息。
  Future<AgentMessage> sendMessage(String threadID, String content);
  /// 启动指定线程的 AI 生成任务。
  Future<AgentRun> startRun(String threadID);
}
