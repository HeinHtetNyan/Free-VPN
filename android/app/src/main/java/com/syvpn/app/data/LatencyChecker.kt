package com.syvpn.app.data

import java.net.InetSocketAddress
import java.net.Socket

/**
 * Approximates round-trip latency to a location's relay by timing a plain
 * TCP connect to it on port 443 — not WireGuard's own (UDP) port. Raw
 * ICMP/UDP RTT measurement isn't reliably available to an unprivileged
 * Android app across devices/carriers, so this measures real network RTT
 * to the same physical relay machine instead, which is what actually
 * matters for "fastest connection point" (docs/MOBILE.md).
 */
object LatencyChecker {
    private const val TIMEOUT_MS = 2500
    private const val PROBE_PORT = 443

    /** Blocking socket I/O — call from a background dispatcher. Null on
     * failure/timeout (e.g. offline, relay unreachable). */
    fun measureMs(host: String): Int? {
        if (host.isBlank()) return null
        return try {
            val start = System.nanoTime()
            Socket().use { socket ->
                socket.connect(InetSocketAddress(host, PROBE_PORT), TIMEOUT_MS)
            }
            ((System.nanoTime() - start) / 1_000_000).toInt()
        } catch (e: Exception) {
            null
        }
    }
}
