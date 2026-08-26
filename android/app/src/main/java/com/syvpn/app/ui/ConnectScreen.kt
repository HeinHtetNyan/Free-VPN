package com.syvpn.app.ui

import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.RadioButton
import androidx.compose.material3.RadioButtonDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.dp
import com.syvpn.app.R
import com.syvpn.app.ads.AdsterraBannerAd
import com.syvpn.app.data.ApiClient
import com.syvpn.app.ui.theme.LocalStatusColors

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
    connectedNowCount: Int?,
    locationLatencies: Map<String, Int?>,
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
                .padding(horizontal = 24.dp, vertical = 20.dp),
            horizontalAlignment = Alignment.Start,
            verticalArrangement = Arrangement.spacedBy(24.dp),
        ) {
            AppHeader(connectedNowCount)

            StatusCard(connectionState)

            val isBusy = connectionState is ConnectionUiState.Connecting
            val isConnected = connectionState is ConnectionUiState.Connected

            Button(
                onClick = { if (isConnected) onDisconnectClick() else onConnectClick() },
                enabled = !isBusy && selectedLocationId != null,
                shape = RoundedCornerShape(16.dp),
                colors = ButtonDefaults.buttonColors(
                    containerColor = MaterialTheme.colorScheme.primary,
                    contentColor = MaterialTheme.colorScheme.onPrimary,
                ),
                modifier = Modifier.fillMaxWidth().height(56.dp),
            ) {
                if (isBusy) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(22.dp),
                        strokeWidth = 2.5.dp,
                        color = MaterialTheme.colorScheme.onPrimary,
                    )
                } else {
                    Text(
                        if (isConnected) "Disconnect" else "Connect",
                        style = MaterialTheme.typography.labelLarge,
                    )
                }
            }

            Text(
                text = "Location is your fastest connection point — it doesn't change your visible country.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )

            Text(
                "CHOOSE A LOCATION",
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )

            LazyColumn(
                modifier = Modifier.fillMaxWidth(),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                items(locations) { location ->
                    LocationRow(
                        location = location,
                        selected = location.id == selectedLocationId,
                        latencyMs = locationLatencies[location.id],
                        onClick = { onSelectLocation(location.id) },
                    )
                }
            }
        }
    }
}

@Composable
private fun AppHeader(connectedNowCount: Int?) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Image(
                painter = painterResource(R.drawable.logo),
                contentDescription = null,
                modifier = Modifier
                    .size(28.dp)
                    .clip(RoundedCornerShape(8.dp)),
            )
            Spacer(modifier = Modifier.width(10.dp))
            Text(
                "SY VPN",
                style = MaterialTheme.typography.titleMedium,
                color = MaterialTheme.colorScheme.onBackground,
            )
        }
        if (connectedNowCount != null) {
            val connected = LocalStatusColors.current.connected
            Row(verticalAlignment = Alignment.CenterVertically) {
                Box(
                    modifier = Modifier
                        .size(6.dp)
                        .background(connected, CircleShape),
                )
                Spacer(modifier = Modifier.width(6.dp))
                Text(
                    "${formatCount(connectedNowCount)} online",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

private fun formatCount(n: Int): String = when {
    n >= 1000 -> "%.1fk".format(n / 1000.0)
    else -> n.toString()
}

@Composable
private fun StatusCard(state: ConnectionUiState) {
    val status = LocalStatusColors.current
    val (label, description, color, pulsing) = when (state) {
        is ConnectionUiState.Idle ->
            StatusVisual("Not connected", "Tap Connect to get started", status.idle, false)
        is ConnectionUiState.Connecting ->
            StatusVisual("Connecting…", "Establishing a secure tunnel", status.connecting, true)
        is ConnectionUiState.Connected ->
            StatusVisual("Connected", "via ${state.locationId.replaceFirstChar { it.uppercase() }}", status.connected, false)
        is ConnectionUiState.Error ->
            StatusVisual("Connection failed", state.message, status.error, false)
    }

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(20.dp))
            .border(1.dp, MaterialTheme.colorScheme.outline, RoundedCornerShape(20.dp))
            .padding(20.dp),
        verticalAlignment = Alignment.Top,
    ) {
        StatusDot(color = color, pulsing = pulsing)
        Spacer(modifier = Modifier.width(14.dp))
        Column {
            Text(
                label,
                style = MaterialTheme.typography.headlineSmall,
                color = MaterialTheme.colorScheme.onSurface,
            )
            if (description.isNotEmpty()) {
                Spacer(modifier = Modifier.height(4.dp))
                Text(
                    description,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

private data class StatusVisual(
    val label: String,
    val description: String,
    val color: Color,
    val pulsing: Boolean,
)

/** A small solid dot for Idle/Connected/Error; a gently pulsing one for
 * Connecting — motion used only where it encodes real state (work in
 * progress), not as decoration. */
@Composable
private fun StatusDot(color: Color, pulsing: Boolean) {
    val alpha = if (pulsing) {
        val transition = rememberInfiniteTransition(label = "status-dot-pulse")
        transition.animateFloat(
            initialValue = 0.35f,
            targetValue = 1f,
            animationSpec = infiniteRepeatable(
                animation = tween(900),
                repeatMode = RepeatMode.Reverse,
            ),
            label = "status-dot-alpha",
        ).value
    } else {
        1f
    }
    Box(
        modifier = Modifier
            .padding(top = 6.dp)
            .size(12.dp)
            .background(color.copy(alpha = alpha), CircleShape),
    )
}

@Composable
private fun LocationRow(location: ApiClient.Location, selected: Boolean, latencyMs: Int?, onClick: () -> Unit) {
    val borderColor = if (selected) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outline
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(MaterialTheme.colorScheme.surfaceVariant, RoundedCornerShape(14.dp))
            .border(1.dp, borderColor, RoundedCornerShape(14.dp))
            .padding(start = 16.dp, end = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        Text(
            location.displayName,
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurface,
            modifier = Modifier.padding(vertical = 14.dp),
        )
        Row(verticalAlignment = Alignment.CenterVertically) {
            LatencyBadge(latencyMs)
            Spacer(modifier = Modifier.width(4.dp))
            RadioButton(
                selected = selected,
                onClick = onClick,
                colors = RadioButtonDefaults.colors(
                    selectedColor = MaterialTheme.colorScheme.primary,
                    unselectedColor = MaterialTheme.colorScheme.onSurfaceVariant,
                ),
            )
        }
    }
}

/** ms == null covers both "not measured yet" and "unreachable" — both show
 * as a neutral dash rather than a false zero or an alarming error state,
 * since a slow/failed latency probe isn't itself a connection problem. */
@Composable
private fun LatencyBadge(latencyMs: Int?) {
    val status = LocalStatusColors.current
    val color = when {
        latencyMs == null -> MaterialTheme.colorScheme.onSurfaceVariant
        latencyMs < 100 -> status.connected
        latencyMs < 250 -> status.connecting
        else -> status.error
    }
    Text(
        text = if (latencyMs != null) "${latencyMs} ms" else "—",
        style = MaterialTheme.typography.bodySmall,
        color = color,
    )
}
