# @file CMakeLists.txt
# @brief voice_input 插件 Android 原生构建（R4：Oboe 采集；R7：Oboe 播放）。
#
# Oboe 经 Gradle prefab 依赖引入（build.gradle.kts 中
# com.google.oboe:oboe:1.9.0），CMake 只做 find_package。
# 共享 C 源（环缓 + ABI 契约）单一来源在
# ../../../lib/features/voice_stream/native/，此处仅 include，不复制。
cmake_minimum_required(VERSION 3.22.1)
project(voice_input LANGUAGES C CXX)

set(NATIVE_SHARED_DIR ../../../lib/features/voice_stream/native)

add_library(voice_input SHARED
  src/main/cpp/voice_input_oboe.cpp
  src/main/cpp/voice_output_oboe.cpp
  ${NATIVE_SHARED_DIR}/spsc_ring_buffer.h
  ${NATIVE_SHARED_DIR}/playback_queue.h
  ${NATIVE_SHARED_DIR}/voice_input.h
  ${NATIVE_SHARED_DIR}/voice_output.h)

target_include_directories(voice_input PRIVATE
  ${CMAKE_CURRENT_SOURCE_DIR}/src/main/cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/${NATIVE_SHARED_DIR})

find_package(oboe REQUIRED CONFIG)
target_link_libraries(voice_input PRIVATE oboe::oboe)

target_compile_features(voice_input PRIVATE cxx_std_17)
target_compile_options(voice_input PRIVATE -Wall -Wextra)
