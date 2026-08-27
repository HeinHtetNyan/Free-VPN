package com.syvpn.app.vpn

import android.content.Context
import android.content.Intent
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import android.net.VpnService
import org.amnezia.awg.backend.GoBackend
import org.amnezia.awg.backend.NoopTunnelActionHandler
import org.amnezia.awg.backend.Tunnel
import org.amnezia.awg.config.Config
import java.io.ByteArrayInputStream

/**
 * Wraps AmneziaWG's Android tunnel library (GoBackend) — the actual tunnel
 * this app drives. A drop-in fork of the official WireGuard Android app's
 * shape (org.amnezia.awg.{backend,config} mirrors
 * com.wireguard.android.backend / com.wireguard.config almost 1:1), needed
 * because the plain WireGuard library can't parse the AmneziaWG obfuscation
 * fields (Jc/Jmin/.../H4) backend/internal/servers/amnezia.go embeds in
 * every client config — see docs/ARCHITECTURE.md "Censorship resistance".
 * Not yet compiled against a real Android SDK in this environment; the API
 * surface here matches com.zaneschepke:amneziawg-android as pinned in
 * app/build.gradle.kts — re-verify against that artifact's docs if Gradle
 * reports it's changed.
 *
 * Typed as the concrete GoBackend, not the Backend interface, specifically
 * for isRunning() below — getRunningTunnelNames() isn't on the Backend
 * interface.
 */
class VpnConnectionManager(context: Context) {
    private val backend: GoBackend = GoBackend(context, NoopTunnelActionHandler())
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

    /** configText is the raw AmneziaWG .conf returned by ApiClient.connect(). */
    fun connect(configText: String) {
        val config = Config.parse(ByteArrayInputStream(configText.toByteArray(Charsets.UTF_8)))
        backend.setState(tunnel, Tunnel.State.UP, config)
    }

    fun disconnect() {
        backend.setState(tunnel, Tunnel.State.DOWN, null)
    }

    fun currentState(): Tunnel.State = backend.getState(tunnel)

    /**
     * GoBackend's own getState()/getRunningTunnelNames() are both dead ends
     * for this: decompiling the library shows getRunningTunnelNames() just
     * wraps the exact same in-memory currentTunnel instance field getState()
     * checks — neither one actually queries the native layer, so neither
     * survives a fresh GoBackend instance (e.g. after this Activity/process
     * is recreated), even though the real tunnel (a system VpnService,
     * independent of our process) is still running. ConnectivityManager is
     * the OS's own ground truth instead — ask it whether this app is
     * *currently* being routed through any VPN, independent of any
     * WireGuard/AmneziaWG-library bookkeeping.
     */
    fun isRunning(context: Context): Boolean {
        val cm = context.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
        val caps = cm.getNetworkCapabilities(cm.activeNetwork)
        return caps?.hasTransport(NetworkCapabilities.TRANSPORT_VPN) == true
    }

    private class AppTunnel(private val onChange: (Tunnel.State) -> Unit) : Tunnel {
        override fun getName(): String = "vpnapp"
        override fun onStateChange(newState: Tunnel.State) = onChange(newState)
        override fun isIpv4ResolutionPreferred(): Boolean = false
        override fun isMetered(): Boolean = false
    }
}
