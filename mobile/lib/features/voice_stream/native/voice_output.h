/// @file voice_output.h
/// @brief 播放端 C ABI 契约：控制面（start/stop）+ 数据入口（write）。
///
/// Android（Oboe，C++ 实现 voice_output_oboe.cpp）与 iOS（RemoteIO 输出侧，
/// ObjC++ 实现 voice_output_ios.mm）共用本契约；Dart 侧 package:voice_input
/// 的 FFI 绑定按同一符号表对接。与采集端（voice_input.h 的 vi_* 一族）并存
/// 于同一原生库，可同时运行（全双工）。
///
/// 数据面：Dart 线程经 vo_write 是唯一生产者（写 playback_queue，全有或
/// 全无），音频回调线程是唯一消费者（acquire/commit 取数，空则输出静音）。
/// 消费者永不阻塞；欠载/拒绝量化见 vo_underrun / vo_dropped。
///
/// @note start/stop 由 Dart 主 isolate 调用；实现内部打开/关闭音频流允许
///       在调用线程做阻塞式初始化，但音频回调内严禁阻塞与分配（消费者只做
///       memcpy/	memset 静音）。
#ifndef VSC_VOICE_OUTPUT_H_
#define VSC_VOICE_OUTPUT_H_

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/// @brief 启动配置。
typedef struct {
  int32_t sample_rate; ///< 请求采样率，单位 Hz（如 16000）；0 = 平台默认。
  int32_t capacity_ms; ///< 播放缓冲容量按毫秒预算（如 1000 = 1 秒）。
  int32_t priming_ms;  ///< 预缓冲阈值（如 40）；0 = 不预缓冲，负值取默认。
} vo_config_t;

/// @brief 打开输出流并启动回调消费播放队列。
///
/// @param cfg 启动配置，不得为空。
/// @return 0 成功；-1 已在运行；-2 流打开失败；-3 参数非法。
int32_t vo_start(const vo_config_t *cfg);

/// @brief 停止播放并释放流；重复调用幂等成功。
///
/// @return 0 成功（含未启动时的幂等调用）。
int32_t vo_stop(void);

/// @brief 拷贝写入待播放 PCM（生产者唯一入口，全有或全无）。
///
/// 缓冲空间不足时整块拒绝并计入 vo_dropped，永不阻塞。
///
/// @param src 待写入 PCM 字节（16 位小端，交错排布）。
/// @param len 待写入长度，必须为正数。
/// @return 实际接受的字节数：len（全部接受）或 0（整块拒绝）。
int32_t vo_write(const uint8_t *src, int32_t len);

/// @brief 当前缓冲中的可播放字节数。
///
/// @return 可播放字节数；未启动时返回 0。
int32_t vo_buffered(void);

/// @brief 欠载静音累计字节数（AudioSink.underrunBytes 的数据来源）。
///
/// @return 自启动以来回调输出静音的字节总数（启动预缓冲期不计）。
uint64_t vo_underrun(void);

/// @brief 空间不足被整块拒绝的累计字节数（播放侧丢帧统计）。
///
/// @return 自启动以来写入被拒绝的字节总数。
uint64_t vo_dropped(void);

/// @brief 回读协商后的实际格式（采样率是请求不是保证，见接口文档 8.1）。
///
/// @param sample_rate 输出实际采样率；未启动时写 0。
/// @param channels 输出声道数；未启动时写 0。
/// @param bits 输出每样本位深；未启动时写 0。
/// @return 0 成功；-1 参数为空。
int32_t vo_format(int32_t *sample_rate, int32_t *channels, int32_t *bits);

#ifdef __cplusplus
}
#endif

#endif // VSC_VOICE_OUTPUT_H_
