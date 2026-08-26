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
}
