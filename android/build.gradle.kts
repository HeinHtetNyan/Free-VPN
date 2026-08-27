plugins {
    // Bumped from 8.5.2: amneziawg-android 2.3.7 pulls in androidx.core-ktx
    // 1.17.0 transitively, which requires compileSdk 36 + AGP 8.9.1+ (see
    // app/build.gradle.kts compileSdk and gradle/wrapper/gradle-wrapper.properties,
    // bumped together with this for the same reason).
    id("com.android.application") version "8.9.1" apply false
    // Bumped from 2.0.20 alongside AGP above: amneziawg-android 2.3.7 pulls
    // in kotlin-stdlib 2.2.21 transitively, which a 2.0.x Kotlin compiler
    // can't read ("compiled with an incompatible version of Kotlin").
    id("org.jetbrains.kotlin.android") version "2.2.21" apply false
    id("org.jetbrains.kotlin.plugin.compose") version "2.2.21" apply false
}
