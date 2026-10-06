package com.syvpn.app

import android.content.Context
import android.content.pm.ApplicationInfo
import android.content.pm.PackageManager
import android.os.Build
import android.os.Debug
import java.io.File
import java.security.MessageDigest

/**
 * Release-only runtime tamper checks. Never active in debug builds.
 *
 * - Debugger attached / app marked debuggable.
 * - Frida (agent/gadget mapped into this process, frida-server on disk).
 * - Xposed/LSPosed hooks (bridge class present, hook frames on the stack).
 * - Signature: our signing cert must match the release/upload cert or the Play
 *   App Signing cert. If the Play cert hash isn't configured at build time and
 *   the app was installed by Google Play, the check is skipped (Play re-signs
 *   with a key we can't know), so legitimate Play users are never blocked.
 *
 * Deliberately NOT checked: root, emulator, developer options (false positives).
 */
object TamperGuard {

    fun isThreatDetected(context: Context): Boolean {
        if (BuildConfig.DEBUG) return false
        return try {
            debuggerAttached(context) || fridaDetected() || xposedDetected() || badSignature(context)
        } catch (_: Throwable) {
            false // never block on an error in the check itself
        }
    }

    private fun debuggerAttached(context: Context): Boolean {
        if (Debug.isDebuggerConnected() || Debug.waitingForDebugger()) return true
        return (context.applicationInfo.flags and ApplicationInfo.FLAG_DEBUGGABLE) != 0
    }

    private fun fridaDetected(): Boolean {
        try {
            File("/proc/self/maps").useLines { lines ->
                for (l in lines) {
                    val s = l.lowercase()
                    if (s.contains("frida-agent") || s.contains("frida-gadget") ||
                        s.contains("libfrida") || s.contains("re.frida.server")
                    ) return true
                }
            }
        } catch (_: Throwable) {
        }
        if (File("/data/local/tmp/frida-server").exists() ||
            File("/data/local/tmp/re.frida.server").exists()
        ) return true
        return Thread.getAllStackTraces().keys.any { it.name == "gum-js-loop" }
    }

    private fun xposedDetected(): Boolean {
        try {
            Class.forName("de.robv.android.xposed.XposedBridge")
            return true
        } catch (_: Throwable) {
        }
        val frames = Throwable().stackTrace
        if (frames.any { it.className.startsWith("de.robv.android.xposed") || it.className.contains("lsposed", true) }) return true
        try {
            File("/proc/self/maps").useLines { lines ->
                for (l in lines) {
                    val s = l.lowercase()
                    if (s.contains("xposedbridge") || s.contains("lsposed") || s.contains("edxp")) return true
                }
            }
        } catch (_: Throwable) {
        }
        return false
    }

    private fun signerHashes(context: Context): Set<String> {
        val pm = context.packageManager
        val sigs = if (Build.VERSION.SDK_INT >= 28) {
            val info = pm.getPackageInfo(context.packageName, PackageManager.GET_SIGNING_CERTIFICATES)
            val si = info.signingInfo ?: return emptySet()
            if (si.hasMultipleSigners()) si.apkContentsSigners else si.signingCertificateHistory
        } else {
            @Suppress("DEPRECATION")
            pm.getPackageInfo(context.packageName, PackageManager.GET_SIGNATURES).signatures
        } ?: return emptySet()
        val md = MessageDigest.getInstance("SHA-256")
        return sigs.map { s -> md.digest(s.toByteArray()).joinToString("") { "%02x".format(it) } }.toSet()
    }

    private fun installedByPlay(context: Context): Boolean {
        val pkg = try {
            if (Build.VERSION.SDK_INT >= 30) {
                context.packageManager.getInstallSourceInfo(context.packageName).installingPackageName
            } else {
                @Suppress("DEPRECATION")
                context.packageManager.getInstallerPackageName(context.packageName)
            }
        } catch (_: Throwable) {
            null
        }
        return pkg == "com.android.vending"
    }

    private fun badSignature(context: Context): Boolean {
        val release = BuildConfig.TAMPER_RELEASE_CERT_SHA256
        val play = BuildConfig.TAMPER_PLAY_CERT_SHA256
        if (release.isEmpty() && play.isEmpty()) return false // nothing to compare against
        val signers = signerHashes(context)
        if (signers.isEmpty()) return false
        if (signers.any { it == release || (play.isNotEmpty() && it == play) }) return false
        // Mismatch. Play-signed installs can't be verified without the Play cert.
        if (play.isEmpty() && installedByPlay(context)) return false
        return true
    }
}
