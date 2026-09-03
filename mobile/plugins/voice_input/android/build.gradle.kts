group = "com.gangcaiyoule.voice_input"
version = "1.0-SNAPSHOT"

buildscript {
    repositories {
        google()
        mavenCentral()
    }

    dependencies {
        classpath("com.android.tools.build:gradle:9.1.0")
    }
}

allprojects {
    repositories {
        google()
        mavenCentral()
    }
}

plugins {
    id("com.android.library")
}

android {
    namespace = "com.gangcaiyoule.voice_input"

    compileSdk = 36

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    defaultConfig {
        minSdk = 24

        externalNativeBuild {
            cmake {
                arguments += listOf("-DANDROID_STL=c++_shared")
            }
        }
    }

    // 纯 FFI 插件（ffiPlugin: true）：无 Kotlin 平台通道代码。
    externalNativeBuild {
        cmake {
            path = file("CMakeLists.txt")
            version = "3.22.1"
        }
    }

    buildFeatures {
        prefab = true
    }

    packaging {
        jniLibs {
            // c++_shared 由各插件自带，避免重复打包冲突时在此 pickFirst。
            keepDebugSymbols += listOf("*/arm64-v8a/*.so", "*/armeabi-v7a/*.so", "*/x86_64/*.so")
        }
    }
}

dependencies {
    // Oboe 1.9.0：prefab 形式提供 CMake 包，R4 采集用。
    implementation("com.google.oboe:oboe:1.9.0")
}
