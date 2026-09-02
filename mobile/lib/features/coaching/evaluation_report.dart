/// Immutable client-side contracts for a completed practice review.
enum EvaluationStatus { queued, running, ready, failed }
enum EvaluationScoreability { provisional, insufficient }
enum EvaluationScoreScale { percentage100, ieltsBand9 }
enum EvaluationFeedbackCategory { correction, strength, recommendedExpression }
enum EvaluationRepracticeMode { none, sameQuestion }

class EvaluationReportDetail {
  const EvaluationReportDetail({
    required this.schemaVersion,
    required this.sceneType,
    required this.summary,
    required this.scoreability,
    required this.questions,
    required this.dimensions,
  });

  final String schemaVersion;
  final String sceneType;
  final String summary;
  final EvaluationScoreability scoreability;
  final List<EvaluationReportQuestion> questions;
  final List<EvaluationReportDimension> dimensions;
}

class EvaluationReportQuestion {
  const EvaluationReportQuestion({required this.id, required this.text, this.transcript});
  final String id;
  final String text;
  final String? transcript;
}

class EvaluationReportDimension {
  const EvaluationReportDimension({required this.key, required this.scale, required this.score, required this.feedback});
  final String key;
  final EvaluationScoreScale scale;
  final double? score;
  final List<EvaluationFeedbackItem> feedback;
}

class EvaluationFeedbackItem {
  const EvaluationFeedbackItem({required this.id, required this.category, required this.recommendation, required this.repracticeMode});
  final String id;
  final EvaluationFeedbackCategory category;
  final String recommendation;
  final EvaluationRepracticeMode repracticeMode;
}
