import 'package:flutter/material.dart';

import 'practice_controller.dart';

final class PracticePage extends StatefulWidget {
  const PracticePage({required this.controller, super.key});
  final PracticeController controller;
  @override
  State<PracticePage> createState() => _PracticePageState();
}

final class _PracticePageState extends State<PracticePage> {
  late final TextEditingController _answerController;
  PracticeController get controller => widget.controller;

  @override
  void initState() {
    super.initState();
    _answerController = TextEditingController();
    controller.addListener(_changed);
    if (controller.state == PracticeViewState.initial) controller.load();
  }

  @override
  void dispose() {
    controller.removeListener(_changed);
    _answerController.dispose();
    super.dispose();
  }

  void _changed() => setState(() {});

  @override
  Widget build(BuildContext context) {
    if (controller.state == PracticeViewState.loading) return const Scaffold(body: Center(child: CircularProgressIndicator()));
    if (controller.state == PracticeViewState.failed) return Scaffold(appBar: AppBar(title: const Text('练习')), body: _message(context));
    final question = controller.currentQuestion;
    if (controller.state == PracticeViewState.completed || (question == null && controller.session?.status == 'COMPLETED')) {
      return Scaffold(appBar: AppBar(title: const Text('练习完成')), body: const Center(child: Text('本次练习已完成。')));
    }
    return Scaffold(
      appBar: AppBar(title: const Text('练习进行中')),
      body: Padding(
        padding: const EdgeInsets.all(20),
        child: question == null
            ? _buildCompletion(context)
            : Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
                Text('问题 ${question.position}', style: Theme.of(context).textTheme.titleMedium),
                const SizedBox(height: 12),
                Text(question.content, style: Theme.of(context).textTheme.headlineSmall),
                const SizedBox(height: 20),
                TextField(controller: _answerController, enabled: !controller.isBusy, minLines: 4, maxLines: 8, decoration: const InputDecoration(border: OutlineInputBorder(), labelText: '你的回答')),
                if (controller.errorMessage != null) Padding(padding: const EdgeInsets.only(top: 8), child: Text(controller.errorMessage!, style: TextStyle(color: Theme.of(context).colorScheme.error))),
                const SizedBox(height: 12),
                FilledButton.icon(onPressed: controller.isBusy ? null : () => controller.submit(_answerController.text), icon: controller.state == PracticeViewState.submitting ? const SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2)) : const Icon(Icons.send), label: const Text('提交回答')),
              ]),
      ),
    );
  }

  Widget _buildCompletion(BuildContext context) => Center(child: Column(mainAxisSize: MainAxisSize.min, children: [
        const Text('所有问题均已回答。'),
        const SizedBox(height: 12),
        if (controller.errorMessage != null) Text(controller.errorMessage!, style: TextStyle(color: Theme.of(context).colorScheme.error)),
        FilledButton(onPressed: controller.isBusy ? null : controller.complete, child: const Text('完成练习')),
      ]));

  Widget _message(BuildContext context) => Center(child: Column(mainAxisSize: MainAxisSize.min, children: [
        Text(controller.errorMessage ?? '练习加载失败。'),
        const SizedBox(height: 12),
        FilledButton(onPressed: controller.retry, child: const Text('重试')),
      ]));
}
