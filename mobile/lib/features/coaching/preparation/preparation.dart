import 'package:flutter/material.dart';

import '../scene/scene.dart';
import 'preparation_controller.dart';

final class PreparationPage extends StatefulWidget {
  const PreparationPage({required this.controller, this.onSelectionComplete, super.key});

  final PreparationController controller;
  final ValueChanged<SceneSelectionSnapshot>? onSelectionComplete;

  @override
  State<PreparationPage> createState() => _PreparationPageState();
}

class _PreparationPageState extends State<PreparationPage> {
  PreparationController get controller => widget.controller;

  @override
  void initState() {
    super.initState();
    controller.addListener(_changed);
    controller.loadIfNeeded();
  }

  @override
  void dispose() {
    controller.removeListener(_changed);
    super.dispose();
  }

  void _changed() => setState(() {});

  @override
  Widget build(BuildContext context) {
    final detail = controller.detail;
    return Scaffold(
      appBar: AppBar(title: const Text('Practice')),
      body: controller.selectedScene == null
          ? _buildCatalog(context)
          : _buildDetail(context, detail),
    );
  }

  Widget _buildCatalog(BuildContext context) {
    if (controller.isLoadingScenes) return const Center(child: CircularProgressIndicator());
    if (controller.state == PreparationViewState.failed) {
      return _MessageState(message: controller.errorMessage!, action: controller.retryLastFailure);
    }
    if (controller.state == PreparationViewState.empty) {
      return const _MessageState(message: '暂无可用练习场景。');
    }
    final groups = <PracticeExperience, List<SceneDefinition>>{};
    for (final scene in controller.scenes) {
      groups.putIfAbsent(scene.experience, () => []).add(scene);
    }
    return ListView(
      padding: const EdgeInsets.all(20),
      children: [
        const Text('选择练习类型', style: TextStyle(fontSize: 24, fontWeight: FontWeight.bold)),
        const SizedBox(height: 20),
        for (final entry in groups.entries) ...[
          Text(entry.key.displayName, style: Theme.of(context).textTheme.titleLarge),
          const SizedBox(height: 8),
          ...entry.value.map((scene) => Card(
                child: ListTile(
                  title: Text(scene.name),
                  subtitle: Text(scene.prompt.publicSceneBrief),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () => controller.selectScene(scene),
                ),
              )),
          const SizedBox(height: 16),
        ],
      ],
    );
  }

  Widget _buildDetail(BuildContext context, SceneDefinition? detail) {
    if (controller.isLoadingDetail) return const Center(child: CircularProgressIndicator());
    if (controller.errorMessage != null) {
      return _MessageState(message: controller.errorMessage!, action: controller.retryLastFailure);
    }
    if (detail == null) return const _MessageState(message: '场景详情暂时不可用。');
    return ListView(
      padding: const EdgeInsets.all(20),
      children: [
        TextButton.icon(onPressed: controller.clearSelection, icon: const Icon(Icons.arrow_back), label: const Text('返回场景列表')),
        Text(detail.name, style: const TextStyle(fontSize: 26, fontWeight: FontWeight.bold)),
        const SizedBox(height: 12),
        _Section(title: '场景描述', body: detail.prompt.publicSceneBrief),
        _Section(title: '练习目标', body: detail.prompt.practiceGoal),
        _Section(title: '角色设定', body: '你：${detail.prompt.userRole}\nAI：${detail.prompt.aiRole}\n${detail.prompt.personaSummary}'),
        _Section(title: '练习重点', body: detail.prompt.focusAreas.join('、')),
        const SizedBox(height: 12),
        const Text('选择用户角色', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
        ...controller.roles.map((role) => RadioListTile<String>(
              value: role.id,
              groupValue: controller.selectedRole?.id,
              title: Text(role.displayName),
              subtitle: Text('${role.responsibilities}\n风格：${role.style}'),
              onChanged: (_) => controller.selectRole(role),
            )),
        const SizedBox(height: 12),
        const Text('选择练习模式', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
        ...controller.availableOptions.map((option) => RadioListTile<String>(
              value: option.id,
              groupValue: controller.selectedOption?.id,
              title: Text('${option.mode.displayName} · ${option.displayName}'),
              subtitle: Text('建议时长：${_duration(option.suggestedDurationSeconds)}'),
              onChanged: (_) => controller.selectOption(option),
            )),
        const SizedBox(height: 16),
        FilledButton.icon(
          onPressed: controller.hasCompleteSelection ? () => _complete(context) : null,
          icon: const Icon(Icons.play_arrow),
          label: const Text('开始练习'),
        ),
      ],
    );
  }

  void _complete(BuildContext context) {
    final result = controller.selectionResult;
    if (result == null) return;
    widget.onSelectionComplete?.call(result);
    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('场景选择已完成，可以进入准备流程。')));
  }
}

final class _Section extends StatelessWidget {
  const _Section({required this.title, required this.body});

  final String title;
  final String body;

  @override
  Widget build(BuildContext context) => Padding(
        padding: const EdgeInsets.only(bottom: 16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(title, style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 4),
          Text(body),
        ]),
      );
}

final class _MessageState extends StatelessWidget {
  const _MessageState({required this.message, this.action});

  final String message;
  final Future<void> Function()? action;

  @override
  Widget build(BuildContext context) => Center(
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          Text(message, textAlign: TextAlign.center),
          if (action != null) ...[
            const SizedBox(height: 12),
            FilledButton.icon(onPressed: action, icon: const Icon(Icons.refresh), label: const Text('重试')),
          ],
        ]),
      );
}

String _duration(int seconds) => '${seconds ~/ 60} 分钟';
