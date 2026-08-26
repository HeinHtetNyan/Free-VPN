package com.syvpn.app.data

import android.content.Context
import java.util.UUID

/**
 * The app's anonymous, device-bound identity — generated once on first
 * launch and persisted locally. Sent to POST /auth/register. See
 * ../../../../docs/BACKEND.md and docs/DECISIONS.md for why this over real
 * accounts: frictionless onboarding, no email required up front.
 */
object DeviceIdentity {
    private const val PREFS_NAME = "device_identity"
    private const val KEY_DEVICE_ID = "device_id"
    private const val KEY_TOKEN = "auth_token"
    private const val KEY_LAST_LOCATION = "last_connected_location_id"
    private const val KEY_LAST_CONFIG = "last_connected_config"

    fun getOrCreateDeviceId(context: Context): String {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        prefs.getString(KEY_DEVICE_ID, null)?.let { return it }

        val newId = UUID.randomUUID().toString()
        prefs.edit().putString(KEY_DEVICE_ID, newId).apply()
        return newId
    }

    fun getStoredToken(context: Context): String? =
        context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE).getString(KEY_TOKEN, null)

    fun storeToken(context: Context, token: String) {
        context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .edit().putString(KEY_TOKEN, token).apply()
    }

    // Purely a UI label ("Connected via X") for restoring the ConnectScreen
    // after MainActivity is recreated while the tunnel (a real system
    // VpnService, independent of the Activity's lifecycle) is still up —
    // not used to decide whether a tunnel is actually running.
    fun getLastConnectedLocationId(context: Context): String? =
        context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE).getString(KEY_LAST_LOCATION, null)

    fun storeLastConnectedLocationId(context: Context, locationId: String) {
        context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .edit().putString(KEY_LAST_LOCATION, locationId).apply()
    }

    // The raw WireGuard .conf last used to connect. Needed to re-attach a
    // real, controllable handle to an already-running tunnel after this
    // app's process was recreated — see VpnConnectionManager.isRunning() and
    // MainActivity.onCreate. Re-supplying the exact same config (same client
    // private key, same peer) to GoBackend.setState(UP) doesn't register a
    // new peer on the backend or create a second interface: the native layer
    // replaces the existing same-named tunnel in place.
    fun getLastConnectedConfig(context: Context): String? =
        context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE).getString(KEY_LAST_CONFIG, null)

    fun storeLastConnectedConfig(context: Context, configText: String) {
        context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .edit().putString(KEY_LAST_CONFIG, configText).apply()
    }
}
