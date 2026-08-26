package com.syvpn.app.vpn

import android.content.Context
import android.content.Intent
import android.net.VpnService
import com.wireguard.android.backend.Backend
import com.wireguard.android.backend.GoBackend
import com.wireguard.android.backend.Tunnel
import com.wireguard.config.Config
import java.io.ByteArrayInputStream

/**
 * Wraps WireGuard's official Android library (GoBackend) — the actual
 * tunnel this app drives. Shape follows WireGuard's own reference Android
 * app. Not yet compiled against a real Android SDK in this environment; the
 * API surface here (GoBackend/Tunnel/Config) matches the
 * com.wireguard.android:tunnel library as of the version pinned in
 * app/build.gradle.kts — re-verify against that artifact's docs if Gradle
 * reports it's changed.
 */
class VpnConnectionManager(context: Context) {
    private val backend: Backend = GoBackend(context)
    private val tunnel = AppTunnel { state -> onStateChange?.invoke(state) }

    /** UI hook — set to update connected/disconnected/error state on screen. */
    var onStateChange: ((Tunnel.State) -> Unit)? = null

    /**
     * Android requires explicit user consent for a VPN service. Call this
     * before connect(); if it returns non-null, launch it for a result
     * (ActivityResultContracts.StartActivityForResult) — a null result means
     * permission was already granted in a previous session.
     */
    fun permissionIntent(context: Context): Intent? = VpnService.prepare(context)

    /** configText is the raw WireGuard .conf returned by ApiClient.connect(). */
    fun connect(configText: String) {
        val config = Config.parse(ByteArrayInputStream(configText.toByteArray(Charsets.UTF_8)))
        backend.setState(tunnel, Tunnel.State.UP, config)
    }

    fun disconnect() {
        backend.setState(tunnel, Tunnel.State.DOWN, null)
    }

    fun currentState(): Tunnel.State = backend.getState(tunnel)

    private class AppTunnel(private val onChange: (Tunnel.State) -> Unit) : Tunnel {
        override fun getName(): String = "vpnapp"
        override fun onStateChange(newState: Tunnel.State) = onChange(newState)
    }
}
