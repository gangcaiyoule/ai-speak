#
# voice_input iOS podspec（R5：RemoteIO 采集 + ObjC++ 会话壳）。
# Run `pod lib lint voice_input.podspec` to validate before publishing.
#
Pod::Spec.new do |s|
  s.name             = 'voice_input'
  s.version          = '0.1.0'
  s.summary          = 'voice_stream 采集/播放端平台实现（RemoteIO）。'
  s.description      = <<-DESC
RemoteIO 输入流 + AVAudioSession 激活壳（采集），RemoteIO 输出侧渲染回调
（播放），共用 SPSC 环缓与 playback_queue 欠载状态机，经 dart:ffi 暴露
vi_* / vo_* C ABI。
                       DESC
  s.homepage         = 'http://example.com'
  s.license          = { :file => '../LICENSE' }
  s.author           = { 'Your Company' => 'email@example.com' }
  s.source           = { :path => '.' }
  s.source_files = 'voice_input/Sources/voice_input/**/*'
  s.dependency 'Flutter'
  s.platform = :ios, '15.0'
  s.frameworks = 'AVFoundation', 'AudioToolbox'

  # 共享 C 源单一来源：环缓 spsc_ring_buffer.h 与 ABI 契约 voice_input.h
  # 位于 voice_stream 模块 native/ 目录，编译期经 header search path 引入，
  # 不复制到插件内（Android 侧 CMake 同理）。
  s.pod_target_xcconfig = {
    'DEFINES_MODULE' => 'YES',
    'EXCLUDED_ARCHS[sdk=iphonesimulator*]' => 'i386',
    'HEADER_SEARCH_PATHS' => '"$(PODS_TARGET_SRCROOT)/../../../lib/features/voice_stream/native" "$(inherited)"',
    'CLANG_CXX_LANGUAGE_STANDARD' => 'c++17',
  }

  # If your plugin requires a privacy manifest, for example if it uses any
  # required reason APIs, update the PrivacyInfo.xcprivacy file to describe your
  # plugin's privacy impact, and then uncomment this line.
  s.resource_bundles = {'voice_input_privacy' => ['voice_input/Sources/voice_input/PrivacyInfo.xcprivacy']}
end
