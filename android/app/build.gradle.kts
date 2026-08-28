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

// Release upload key — a real secret, never committed. Kept in
// android/keystore.properties (gitignored, see keystore.properties.example)
// rather than hardcoded like the debug config above.
val keystoreProperties = Properties().apply {
    val file = rootProject.file("keystore.properties")
    if (file.exists()) file.inputStream().use { load(it) }
}
val hasReleaseSigning = keystoreProperties.getProperty("storePassword") != null

android {
    namespace = "com.syvpn.app"
    // Bumped from 35: amneziawg-android 2.3.7's androidx.core-ktx 1.17.0
    // transitive dependency requires compileSdk 36 — see build.gradle.kts.
    compileSdk = 36

    // Also renames the bundleRelease/.aab output (app-release.aab ->
    // sy-vpn-release.aab) — the APK rename below needs the extra
    // applicationVariants hook since AGP ignores archivesName for APKs.
    base.archivesName.set("sy-vpn")

    // amneziawg-android's okhttp3 (DoH resolver) and jspecify both ship an
    // identical multi-release-jar manifest at this path — arbitrarily pick
    // one rather than failing the merge, the content doesn't differ in any
    // way this app depends on.
    packaging {
        resources {
            pickFirsts += "META-INF/versions/9/OSGI-INF/MANIFEST.MF"
        }
    }

    defaultConfig {
        applicationId = "com.syvpn.app"
        minSdk = 26 // VpnService + WireGuard tunnel library both fine at this floor
        targetSdk = 36
        versionCode = 3
        versionName = "0.1.2"

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
        if (hasReleaseSigning) {
            create("release") {
                storeFile = rootProject.file(keystoreProperties.getProperty("storeFile"))
                storePassword = keystoreProperties.getProperty("storePassword")
                keyAlias = keystoreProperties.getProperty("keyAlias")
                keyPassword = keystoreProperties.getProperty("keyPassword")
            }
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
            if (hasReleaseSigning) {
                signingConfig = signingConfigs.getByName("release")
            }
        }
    }

    // Default AGP output naming is "<gradle module name>-<buildType>.apk" —
    // this module is literally named "app" (see settings.gradle.kts), so
    // without this it's "app-debug.apk"/"app-release.apk" regardless of the
    // app's display name or package. Rename the actual output file instead.
    applicationVariants.all {
        outputs.all {
            val output = this as com.android.build.gradle.internal.api.BaseVariantOutputImpl
            output.outputFileName = "sy-vpn-${versionName}-${name}.apk"
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

    // AmneziaWG Android tunnel library — a drop-in fork of the official
    // WireGuard one (same org.amnezia.awg.{backend,config} shape as
    // com.wireguard.android.backend / com.wireguard.config) that understands
    // the obfuscation params backend/internal/servers/amnezia.go embeds in
    // every client config. See docs/ARCHITECTURE.md "Censorship resistance"
    // and ../../docs/MOBILE.md for why this over a community Flutter plugin.
    // Was com.wireguard.android:tunnel:1.0.20230706 (plain WireGuard) before.
    implementation("com.zaneschepke:amneziawg-android:2.3.7")

    debugImplementation("androidx.compose.ui:ui-tooling")
}
