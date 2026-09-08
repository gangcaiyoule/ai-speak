## 0.1.0

* R4/R5：采集端平台实现——Oboe 输入流（Android）/ RemoteIO 输入流 + 会话壳（iOS），vi_* C ABI。
* R7：播放端平台实现——Oboe 输出流（Android）/ RemoteIO 输出侧（iOS），vo_* C ABI；
  欠载策略（预缓冲/迟滞）与丢帧统计在原生 playback_queue 状态机内完成。
* 绑定层条件导入重构：原生平台走 FFI（bindings_native.dart），web 平台走
  抛错桩（bindings_unsupported.dart）；公共契约抽至 bindings_common.dart。
* FFI 符号查找改为惰性（late final）：无原生库的宿主（VM 单测）构造绑定
  不再抛查找失败；下游统一经 createDefault*Bindings 工厂获取绑定。

## 0.0.1

* 初始骨架。
