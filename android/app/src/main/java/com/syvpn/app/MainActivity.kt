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
import com.syvpn.app.data.DeviceIdentity
import com.syvpn.app.ui.ConnectionUiState
import com.syvpn.app.ui.ConnectScreen
import com.syvpn.app.ui.theme.VpnAppTheme
import com.syvpn.app.vpn.VpnConnectionManager
import com.wireguard.android.backend.Tunnel
import kotlinx.coroutines.Dispatchers
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
    private var pendingConfig: String? = null

    private val vpnPermissionLauncher = registerForActivityResult(
        ActivityResultContracts.StartActivityForResult(),
    ) { result ->
        val config = pendingConfig
        val locationId = selectedLocationId
        pendingConfig = null
        if (result.resultCode == RESULT_OK && config != null) {
            lifecycleScope.launch(Dispatchers.IO) {
                try {
                    vpnManager.connect(config)
                    DeviceIdentity.storeLastConnectedConfig(this@MainActivity, config)
                    if (locationId != null) {
                        DeviceIdentity.storeLastConnectedLocationId(this@MainActivity, locationId)
                    }
                    launch(Dispatchers.Main) {
                        connectionState = ConnectionUiState.Connected(locationId ?: "")
                    }
                } catch (e: Exception) {
                    launch(Dispatchers.Main) {
                        connectionState = ConnectionUiState.Error(e.message ?: e.toString())
                    }
                }
            }
        } else {
            connectionState = ConnectionUiState.Error("VPN permission denied")
        }
    }

    private var locations by mutableStateOf<List<ApiClient.Location>>(emptyList())
    private var selectedLocationId by mutableStateOf<String?>(null)
    private var connectionState by mutableStateOf<ConnectionUiState>(ConnectionUiState.Idle)

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        apiClient = ApiClient(ApiClient.DEV_BASE_URL)
        vpnManager = VpnConnectionManager(this)

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
            } catch (e: Exception) {
                launch(Dispatchers.Main) {
                    connectionState = ConnectionUiState.Error("Could not load locations: ${e.message}")
                }
            }
        }
    }

    private fun onConnectClick() {
        val token = authToken ?: return
        val locationId = selectedLocationId ?: return
        connectionState = ConnectionUiState.Connecting

        lifecycleScope.launch(Dispatchers.IO) {
            try {
                val result = apiClient.connect(token, locationId)
                val permissionIntent: Intent? = vpnManager.permissionIntent(this@MainActivity)
                if (permissionIntent != null) {
                    // Not connected yet — still waiting on the user to grant
                    // VPN permission in the system dialog. vpnPermissionLauncher's
                    // callback drives the actual connect() + Connected state
                    // from here once (if) that's granted.
                    pendingConfig = result.config
                    launch(Dispatchers.Main) { vpnPermissionLauncher.launch(permissionIntent) }
                } else {
                    vpnManager.connect(result.config)
                    DeviceIdentity.storeLastConnectedConfig(this@MainActivity, result.config)
                    DeviceIdentity.storeLastConnectedLocationId(this@MainActivity, locationId)
                    launch(Dispatchers.Main) {
                        connectionState = ConnectionUiState.Connected(locationId)
                    }
                }
            } catch (e: Exception) {
                launch(Dispatchers.Main) {
                    connectionState = ConnectionUiState.Error(e.message ?: e.toString())
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
