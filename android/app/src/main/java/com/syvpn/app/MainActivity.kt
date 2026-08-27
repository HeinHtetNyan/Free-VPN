package com.syvpn.app

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.lifecycleScope
import com.syvpn.app.data.ApiClient
import com.syvpn.app.data.DeviceContext
import com.syvpn.app.data.DeviceIdentity
import com.syvpn.app.data.LatencyChecker
import com.syvpn.app.ui.ConnectionUiState
import com.syvpn.app.ui.ConnectScreen
import com.syvpn.app.ui.ReportUiState
import com.syvpn.app.ui.theme.VpnAppTheme
import com.syvpn.app.vpn.VpnConnectionManager
import org.amnezia.awg.backend.Tunnel
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * Wires the thin-client pieces together: DeviceIdentity (local anonymous ID)
 * -> ApiClient (backend/) -> VpnConnectionManager (WireGuard tunnel). See
 * docs/MOBILE.md for why the app itself stays this thin.
 */
class MainActivity : ComponentActivity() {

    private lateinit var apiClient: ApiClient
    private lateinit var vpnManager: VpnConnectionManager
    private var authToken: String? = null

    private val vpnPermissionLauncher = registerForActivityResult(
        ActivityResultContracts.StartActivityForResult(),
    ) { result ->
        if (result.resultCode == RESULT_OK) {
            // Permission granted — only now is it worth asking the server
            // for a peer config; see performConnect().
            performConnect()
        } else {
            connectionState = ConnectionUiState.Error("VPN permission denied")
        }
    }

    private var locations by mutableStateOf<List<ApiClient.Location>>(emptyList())
    private var selectedLocationId by mutableStateOf<String?>(null)
    private var connectionState by mutableStateOf<ConnectionUiState>(ConnectionUiState.Idle)
    private var connectedNowCount by mutableStateOf<Int?>(null)
    private var locationLatencies by mutableStateOf<Map<String, Int?>>(emptyMap())
    private var reportState by mutableStateOf<ReportUiState>(ReportUiState.Idle)
    private lateinit var deviceContext: DeviceContext

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        apiClient = ApiClient(ApiClient.DEV_BASE_URL)
        vpnManager = VpnConnectionManager(this)
        // Captured once — carrier/device/OS don't change mid-session, and
        // capturing fresh on every report would just repeat the same work.
        deviceContext = DeviceContext.capture(this)

        // The tunnel is a real system VpnService, independent of this
        // Activity's lifecycle — it keeps running (or not) across
        // MainActivity being destroyed/recreated (e.g. the app being swiped
        // away in the recents switcher, or a config change). isRunning()
        // asks Android's ConnectivityManager directly instead of trusting
        // this fresh GoBackend instance's own empty in-memory state — see
        // VpnConnectionManager.isRunning() for why currentState() can't be
        // used for this. Re-supplying the cached config re-attaches a real,
        // controllable handle to that same tunnel (see
        // DeviceIdentity.getLastConnectedConfig) — without this, the label
        // would be accurate but Disconnect wouldn't actually be able to
        // close the real tunnel afterward.
        if (vpnManager.isRunning(this)) {
            val lastLocationId = DeviceIdentity.getLastConnectedLocationId(this) ?: ""
            connectionState = ConnectionUiState.Connected(lastLocationId)
            DeviceIdentity.getLastConnectedConfig(this)?.let { cachedConfig ->
                lifecycleScope.launch(Dispatchers.IO) {
                    try {
                        vpnManager.connect(cachedConfig)
                    } catch (e: Exception) {
                        launch(Dispatchers.Main) {
                            connectionState = ConnectionUiState.Error(e.message ?: e.toString())
                        }
                    }
                }
            }
        }
        // Keeps connectionState in sync with any FUTURE state change too,
        // not just the state at launch (e.g. the system tearing the tunnel
        // down for its own reasons while this screen is open).
        vpnManager.onStateChange = { state ->
            runOnUiThread {
                connectionState = when (state) {
                    Tunnel.State.UP -> ConnectionUiState.Connected(
                        DeviceIdentity.getLastConnectedLocationId(this) ?: "",
                    )
                    Tunnel.State.DOWN -> ConnectionUiState.Idle
                    else -> connectionState
                }
            }
        }

        setContent {
            VpnAppTheme {
                ConnectScreen(
                    locations = locations,
                    selectedLocationId = selectedLocationId,
                    onSelectLocation = { selectedLocationId = it },
                    connectionState = connectionState,
                    connectedNowCount = connectedNowCount,
                    locationLatencies = locationLatencies,
                    reportState = reportState,
                    deviceContext = deviceContext,
                    onSubmitReport = ::onSubmitReport,
                    onDismissReport = { reportState = ReportUiState.Idle },
                    onConnectClick = ::onConnectClick,
                    onDisconnectClick = ::onDisconnectClick,
                )
            }
        }

        loadLocations()
    }

    private fun loadLocations() {
        lifecycleScope.launch(Dispatchers.IO) {
            try {
                val token = authToken ?: DeviceIdentity.getStoredToken(this@MainActivity)
                    ?: apiClient.register(DeviceIdentity.getOrCreateDeviceId(this@MainActivity))
                        .also { DeviceIdentity.storeToken(this@MainActivity, it) }
                authToken = token

                val fetched = apiClient.listLocations(token)
                launch(Dispatchers.Main) {
                    locations = fetched
                    selectedLocationId = fetched.firstOrNull()?.id
                }
                measureLatencies(fetched)
                startStatsRefreshLoop(token)
            } catch (e: Exception) {
                launch(Dispatchers.Main) {
                    connectionState = ConnectionUiState.Error("Could not load locations: ${e.message}")
                }
            }
        }
    }

    /** Refreshes the "connected now" count and each location's latency
     * periodically for as long as this screen exists — both are meant to
     * read as live indicators, not a one-time snapshot from app launch. */
    private suspend fun startStatsRefreshLoop(token: String) {
        while (true) {
            try {
                val count = apiClient.connectedNow(token)
                lifecycleScope.launch(Dispatchers.Main) { connectedNowCount = count }
            } catch (e: Exception) {
                // Non-critical — leave the last known count showing rather
                // than surfacing an error for a purely informational stat.
            }
            delay(20_000)
            measureLatencies(locations)
        }
    }

    private suspend fun measureLatencies(locs: List<ApiClient.Location>) {
        val results = locs.map { loc ->
            lifecycleScope.async(Dispatchers.IO) { loc.id to LatencyChecker.measureMs(loc.relayHost) }
        }.awaitAll()
        lifecycleScope.launch(Dispatchers.Main) { locationLatencies = results.toMap() }
    }

    private fun onConnectClick() {
        if (authToken == null || selectedLocationId == null) return
        connectionState = ConnectionUiState.Connecting

        // Ask for VPN permission (if not already granted) before ever
        // asking the server for a peer config — asking the server first
        // and then having the user deny the system dialog just mints a
        // peer that's never used, which is how the server ended up with
        // dozens of never-connected ghost peers (see docs/DECISIONS.md).
        val permissionIntent: Intent? = vpnManager.permissionIntent(this)
        if (permissionIntent != null) {
            // performConnect() runs from vpnPermissionLauncher's callback
            // once (if) the user grants it.
            vpnPermissionLauncher.launch(permissionIntent)
        } else {
            performConnect()
        }
    }

    /** Only ever called once VPN permission is confirmed granted (either
     * already held, or just approved via vpnPermissionLauncher) — safe to
     * ask the server for a peer config here since it's actually going to
     * be used. */
    private fun performConnect() {
        val token = authToken ?: return
        val locationId = selectedLocationId ?: return

        lifecycleScope.launch(Dispatchers.IO) {
            try {
                val result = apiClient.connect(token, locationId)
                vpnManager.connect(result.config)
                DeviceIdentity.storeLastConnectedConfig(this@MainActivity, result.config)
                DeviceIdentity.storeLastConnectedLocationId(this@MainActivity, locationId)
                launch(Dispatchers.Main) {
                    connectionState = ConnectionUiState.Connected(locationId)
                }
            } catch (e: Exception) {
                launch(Dispatchers.Main) {
                    connectionState = ConnectionUiState.Error(e.message ?: e.toString())
                }
            }
        }
    }

    private fun onSubmitReport(message: String, carrier: String) {
        val token = authToken ?: run {
            reportState = ReportUiState.Error("Not ready yet — try again in a moment")
            return
        }
        reportState = ReportUiState.Sending
        // carrier is the user-confirmed/corrected chip selection, not
        // necessarily what auto-detection found — it's the one field the
        // whole feature depends on being accurate, so it overrides
        // deviceContext's raw auto-detected ispName rather than the reverse.
        val context = deviceContext.copy(ispName = carrier)
        lifecycleScope.launch(Dispatchers.IO) {
            try {
                apiClient.submitReport(token, message.trim(), context)
                launch(Dispatchers.Main) { reportState = ReportUiState.Sent }
            } catch (e: Exception) {
                launch(Dispatchers.Main) {
                    reportState = ReportUiState.Error(e.message ?: "Could not send — try again")
                }
            }
        }
    }

    private fun onDisconnectClick() {
        lifecycleScope.launch(Dispatchers.IO) {
            try {
                vpnManager.disconnect()
                launch(Dispatchers.Main) { connectionState = ConnectionUiState.Idle }
            } catch (e: Exception) {
                launch(Dispatchers.Main) {
                    connectionState = ConnectionUiState.Error(e.message ?: e.toString())
                }
            }
        }
    }
}
