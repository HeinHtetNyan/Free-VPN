package com.syvpn.app.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.syvpn.app.ads.AdsterraBannerAd
import com.syvpn.app.data.ApiClient

sealed interface ConnectionUiState {
    data object Idle : ConnectionUiState
    data object Connecting : ConnectionUiState
    data class Connected(val locationId: String) : ConnectionUiState
    data class Error(val message: String) : ConnectionUiState
}

/**
 * The whole first-run/core flow, per docs/MOBILE.md's UX priorities:
 * one-tap connect, a plain-language location picker (no IPs/protocol
 * jargon), and an unambiguous connection-state indicator. Ads never sit on
 * top of the connect button or navigation (docs/PLAY_STORE_COMPLIANCE.md).
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ConnectScreen(
    locations: List<ApiClient.Location>,
    selectedLocationId: String?,
    onSelectLocation: (String) -> Unit,
    connectionState: ConnectionUiState,
    onConnectClick: () -> Unit,
    onDisconnectClick: () -> Unit,
) {
    Scaffold(
        bottomBar = { AdsterraBannerAd(modifier = Modifier.fillMaxWidth()) },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(24.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(20.dp),
        ) {
            StatusIndicator(connectionState)

            Spacer(modifier = Modifier.height(8.dp))

            val isBusy = connectionState is ConnectionUiState.Connecting
            val isConnected = connectionState is ConnectionUiState.Connected

            Button(
                onClick = { if (isConnected) onDisconnectClick() else onConnectClick() },
                enabled = !isBusy && selectedLocationId != null,
                modifier = Modifier.fillMaxWidth().height(56.dp),
            ) {
                if (isBusy) {
                    CircularProgressIndicator(modifier = Modifier.height(24.dp))
                } else {
                    Text(if (isConnected) "Disconnect" else "Connect")
                }
            }

            Text(
                text = "Location is your fastest connection point — it doesn't change your visible country.",
                style = androidx.compose.material3.MaterialTheme.typography.bodySmall,
            )

            Text("Choose a location", style = androidx.compose.material3.MaterialTheme.typography.titleMedium)

            LazyColumn(modifier = Modifier.fillMaxWidth()) {
                items(locations) { location ->
                    LocationRow(
                        location = location,
                        selected = location.id == selectedLocationId,
                        onClick = { onSelectLocation(location.id) },
                    )
                }
            }
        }
    }
}

@Composable
private fun StatusIndicator(state: ConnectionUiState) {
    val (label, description) = when (state) {
        is ConnectionUiState.Idle -> "Not connected" to "Tap Connect to get started"
        is ConnectionUiState.Connecting -> "Connecting…" to ""
        is ConnectionUiState.Connected -> "Connected" to "via ${state.locationId}"
        is ConnectionUiState.Error -> "Connection failed" to state.message
    }
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(modifier = Modifier.padding(16.dp)) {
            Text(label, style = androidx.compose.material3.MaterialTheme.typography.headlineSmall)
            if (description.isNotEmpty()) Text(description)
        }
    }
}

@Composable
private fun LocationRow(location: ApiClient.Location, selected: Boolean, onClick: () -> Unit) {
    Card(modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
        androidx.compose.foundation.layout.Row(
            modifier = Modifier.fillMaxWidth().padding(8.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text(location.displayName, modifier = Modifier.padding(8.dp))
            RadioButton(selected = selected, onClick = onClick)
        }
    }
}
