package com.syvpn.app.data

import android.content.Context
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import android.os.Build
import android.telephony.TelephonyManager
import com.syvpn.app.BuildConfig

/** Technical context auto-attached to an issue report — the actual point of
 * the report feature: telling "MPT blocks this" from "works fine on Ooredoo"
 * apart needs the carrier name, not just free text a user might not think to
 * include themselves (see docs/DECISIONS.md, ISP-level blocking in Myanmar). */
data class DeviceContext(
    val ispName: String,
    val networkType: String,
    val deviceModel: String,
    val osVersion: String,
    val appVersion: String,
) {
    companion object {
        fun capture(context: Context): DeviceContext {
            val cm = context.getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager
            val caps = cm?.let { it.getNetworkCapabilities(it.activeNetwork) }
            val networkType = when {
                caps == null -> "unknown"
                caps.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) -> "wifi"
                caps.hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR) -> "mobile"
                caps.hasTransport(NetworkCapabilities.TRANSPORT_VPN) -> "vpn"
                else -> "other"
            }

            // Carrier/SIM name — getNetworkOperatorName() needs no special
            // permission (unlike device/subscriber IDs), but wrapped
            // defensively anyway since telephony behavior varies by OEM.
            val ispName = try {
                val tm = context.getSystemService(Context.TELEPHONY_SERVICE) as? TelephonyManager
                tm?.networkOperatorName?.takeIf { it.isNotBlank() } ?: ""
            } catch (e: Exception) {
                ""
            }

            return DeviceContext(
                ispName = ispName,
                networkType = networkType,
                deviceModel = "${Build.MANUFACTURER} ${Build.MODEL}",
                osVersion = Build.VERSION.RELEASE ?: "",
                appVersion = BuildConfig.VERSION_NAME,
            )
        }
    }
}
