import java.util.Properties

plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
    id("org.jetbrains.kotlin.plugin.compose")
}

// Adsterra zone IDs are real publisher credentials — kept out of source and
// out of git via android/local.properties (gitignored), not hardcoded.
// See android/local.properties.example. Falls back to a placeholder so the
// app still builds before that file exists.
val localProperties = Properties().apply {
    val file = rootProject.file("local.properties")
    if (file.exists()) file.inputStream().use { load(it) }
}
fun adsterraProperty(key: String): String =
    (localProperties.getProperty(key) ?: System.getenv(key) ?: "ADSTERRA_ZONE_ID_PLACEHOLDER")

android {
    namespace = "com.syvpn.app"
    compileSdk = 35

    defaultConfig {
        applicationId = "com.syvpn.app"
        minSdk = 26 // VpnService + WireGuard tunnel library both fine at this floor
        targetSdk = 35
        versionCode = 1
        versionName = "0.1.0"

        buildConfigField(
            "String",
            "ADSTERRA_BANNER_ZONE_ID",
            "\"${adsterraProperty("ADSTERRA_BANNER_ZONE_ID")}\"",
        )
    }

    signingConfigs {
        getByName("debug") {
            // Pinned to a committed keystore instead of each environment's
            // (or each ephemeral Docker build's) auto-generated one — a
            // debug build from Android Studio and one from the Docker
            // pipeline must produce the SAME signature, or installing one
            // over the other fails with INSTALL_FAILED_UPDATE_INCOMPATIBLE.
            // Standard well-known debug alias/passwords, not a secret.
            storeFile = file("debug.keystore")
            storePassword = "android"
            keyAlias = "androiddebugkey"
            keyPassword = "android"
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    kotlinOptions {
        jvmTarget = "17"
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }
}

dependencies {
    implementation(platform("androidx.compose:compose-bom:2024.09.00"))
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.ui:ui-tooling-preview")
    implementation("androidx.activity:activity-compose:1.9.2")
    implementation("androidx.core:core-ktx:1.13.1")
    implementation("androidx.lifecycle:lifecycle-runtime-ktx:2.8.6")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.8.1")

    // Official WireGuard Android tunnel library — see ../../docs/MOBILE.md
    // for why this over a community Flutter plugin.
    implementation("com.wireguard.android:tunnel:1.0.20230706")

    debugImplementation("androidx.compose.ui:ui-tooling")
}
