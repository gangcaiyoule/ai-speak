/// @file voice_input.h
/// @brief 采集端 C ABI 契约：控制面（start/stop）+ 数据出口（read）。
///
/// Android（Oboe，C++ 实现 voice_input_oboe.cpp）与 iOS（RemoteIO，
/// ObjC++ 实现 voice_input_ios.mm）共用本契约；Dart 侧
/// package:voice_input 的 FFI 绑定按同一符号表对接。
///
/// 数据面：音频回调线程是唯一生产者（写 vsc_spsc_ring_t，丢旧语义），
/// Dart 侧 vi_read 是唯一消费者（拷贝读）。生产者永不阻塞，实时约束
/// 见 spsc_ring_buffer.h。
///
/// @note start/stop 由 Dart 主 isolate 调用；实现内部打开/关闭音频流
///       允许在调用线程做阻塞式初始化，但音频回调内严禁阻塞与分配
///       （生产端只做 memcpy 进环缓）。
#ifndef VSC_VOICE_INPUT_H_
#define VSC_VOICE_INPUT_H_

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/// @brief 启动配置。
typedef struct {
  int32_t sample_rate; ///< 请求采样率，单位 Hz（如 16000）；0 = 平台默认。
  int32_t capacity_ms; ///< 环缓容量按毫秒预算（如 2000 = 2 秒缓冲）。
} vi_config_t;

/// @brief 打开输入流并启动回调写入环缓。
///
/// @param cfg 启动配置，不得为空。
/// @return 0 成功；-1 已在运行；-2 流打开失败；-3 参数非法。
int32_t vi_start(const vi_config_t *cfg);

/// @brief 停止采集并释放流；重复调用幂等成功。
///
/// @return 0 成功（含未启动时的幂等调用）。
int32_t vi_stop(void);

/// @brief 拷贝读出已采集的 PCM 字节（消费者唯一出口）。
///
/// @param dst 目标缓冲，不得为空。
/// @param max_bytes 目标缓冲容量，必须为正数。
/// @return 实际读出字节数（0 表示当前无数据）。
int32_t vi_read(uint8_t *dst, int32_t max_bytes);

/// @brief 环缓累计丢旧字节数（上游 drop 观测口，供 gap 标志推算）。
///
/// @return 自启动以来覆盖丢弃的字节总数。
uint64_t vi_dropped(void);

/// @brief 回读协商后的实际格式（采样率是请求不是保证，见接口文档 8.1）。
///
/// @param sample_rate 输出实际采样率；未启动时写 0。
/// @param channels 输出声道数；未启动时写 0。
/// @param bits 输出每样本位深；未启动时写 0。
/// @return 0 成功；-1 参数为空。
int32_t vi_format(int32_t *sample_rate, int32_t *channels, int32_t *bits);

#ifdef __cplusplus
}
#endif

#endif // VSC_VOICE_INPUT_H_
