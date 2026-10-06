# Hardening rules (R8 full minify+shrink is enabled in build.gradle.kts).

# Aggressive renaming: flatten all obfuscated classes into one package, widen
# access so R8 can merge/inline more, extra optimisation passes.
-repackageclasses ''
-allowaccessmodification
-optimizationpasses 5

# Hide original file names in stack traces (mapping.txt still restores them).
-renamesourcefileattribute SourceFile
-keepattributes SourceFile,LineNumberTable

# Strip debug logging from release builds (no side effects relied upon).
-assumenosideeffects class android.util.Log {
    public static int v(...);
    public static int d(...);
    public static int i(...);
}

# JNI: native methods are bound by exact class+method name, never rename them.
-keepclasseswithmembernames,includedescriptorclasses class * {
    native <methods>;
}

# AmneziaWG tunnel library: Go/JNI backend binds Java classes/methods by name.
# It is open source, so keeping its names costs nothing and keeps the VPN safe.
-keep class org.amnezia.awg.** { *; }
-dontwarn org.amnezia.awg.**
