package com.syvpn.app.ui

import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.BorderStroke
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
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.RadioButton
import androidx.compose.material3.RadioButtonDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import com.syvpn.app.BuildConfig
import com.syvpn.app.R
import com.syvpn.app.ads.AdsterraBannerAd
import com.syvpn.app.data.ApiClient
import com.syvpn.app.data.DeviceContext
import com.syvpn.app.ui.theme.DarkBackground
import com.syvpn.app.ui.theme.LocalStatusColors
import kotlinx.coroutines.delay

sealed interface ReportUiState {
    data object Idle : ReportUiState
    data object Sending : ReportUiState
    data object Sent : ReportUiState
    data class Error(val message: String) : ReportUiState
}

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
    reportState: ReportUiState,
    deviceContext: DeviceContext,
    onSubmitReport: (String, String) -> Unit,
    onDismissReport: () -> Unit,
    onConnectClick: () -> Unit,
    onDisconnectClick: () -> Unit,
) {
    // Inserting the ad's native WebView into the Compose tree costs a
    // layout pass that briefly blanks the whole window (see
    // AdsterraBannerAd's docs on the underlying WebView quirk) — delaying
    // that insertion until just after the real screen has already drawn
    // means that hitch lands on an empty placeholder, not on the screen the
    // user is watching open.
    var showAd by remember { mutableStateOf(false) }
    LaunchedEffect(Unit) {
        delay(50)
        showAd = true
    }

    Scaffold(
        bottomBar = {
            if (showAd) {
                AdsterraBannerAd(modifier = Modifier.fillMaxWidth())
            } else {
                Spacer(modifier = Modifier.fillMaxWidth().height(50.dp).background(DarkBackground))
            }
        },
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

            ReportIssueEntry(reportState, deviceContext, onSubmitReport, onDismissReport)

            Spacer(modifier = Modifier.weight(1f))

            // Sourced from BuildConfig.VERSION_NAME (android/app/build.gradle.kts
            // "versionName") — the one place that ever needs editing; this and
            // every report's device-info payload (DeviceContext) both read the
            // same value, so they can't drift apart.
            Text(
                text = "v${BuildConfig.VERSION_NAME}",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.align(Alignment.CenterHorizontally),
            )
        }
    }
}

/** Report an issue — the app's feedback channel for things server-side
 * metrics can't see, chiefly "this ISP/carrier blocks the VPN outright"
 * (a connection an ISP blocks never even registers as a WireGuard peer).
 * Carrier is a plain typed field — Myanmar has too many ISPs/MVNOs (MPT,
 * Ooredoo, ATOM, Mytel, and various resold/roaming names) to enumerate as
 * fixed options, so the user just types what they're on. */
@Composable
private fun ReportIssueEntry(
    reportState: ReportUiState,
    deviceContext: DeviceContext,
    onSubmit: (String, String) -> Unit,
    onDismiss: () -> Unit,
) {
    var dialogOpen by remember { mutableStateOf(false) }
    var message by remember { mutableStateOf("") }
    var carrierText by remember { mutableStateOf("") }

    OutlinedButton(
        onClick = { dialogOpen = true },
        modifier = Modifier.fillMaxWidth().height(48.dp),
        shape = RoundedCornerShape(14.dp),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline),
        colors = ButtonDefaults.outlinedButtonColors(
            contentColor = MaterialTheme.colorScheme.onSurface,
        ),
    ) {
        Text(
            "Report an issue",
            style = MaterialTheme.typography.labelLarge,
        )
    }

    LaunchedEffect(reportState) {
        if (reportState is ReportUiState.Sent) {
            message = ""
            carrierText = ""
            dialogOpen = false
        }
    }

    if (dialogOpen) {
        Dialog(onDismissRequest = { dialogOpen = false; onDismiss() }) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(20.dp))
                    .padding(20.dp),
                verticalArrangement = Arrangement.spacedBy(14.dp),
            ) {
                Text(
                    "Report an issue",
                    style = MaterialTheme.typography.titleMedium,
                    color = MaterialTheme.colorScheme.onSurface,
                )

                Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text(
                        "NETWORK / CARRIER",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    OutlinedTextField(
                        value = carrierText,
                        onValueChange = { carrierText = it },
                        modifier = Modifier.fillMaxWidth(),
                        placeholder = { Text("e.g. MPT, Ooredoo, ATOM, Mytel, Wi-Fi…") },
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Default),
                        colors = OutlinedTextFieldDefaults.colors(
                            focusedBorderColor = MaterialTheme.colorScheme.primary,
                            unfocusedBorderColor = MaterialTheme.colorScheme.outline,
                            focusedTextColor = MaterialTheme.colorScheme.onSurface,
                            unfocusedTextColor = MaterialTheme.colorScheme.onSurface,
                        ),
                    )
                }

                Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text(
                        "DEVICE",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .background(MaterialTheme.colorScheme.surfaceVariant, RoundedCornerShape(12.dp))
                            .padding(horizontal = 14.dp, vertical = 10.dp),
                    ) {
                        Text(
                            "${deviceContext.deviceModel} · Android ${deviceContext.osVersion} · App v${deviceContext.appVersion}",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurface,
                        )
                    }
                }

                Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text(
                        "WHAT HAPPENED",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    OutlinedTextField(
                        value = message,
                        onValueChange = { message = it },
                        modifier = Modifier.fillMaxWidth(),
                        placeholder = { Text("e.g. \"Won't connect at all\", \"Connects but very slow\"…") },
                        minLines = 4,
                        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Default),
                        colors = OutlinedTextFieldDefaults.colors(
                            focusedBorderColor = MaterialTheme.colorScheme.primary,
                            unfocusedBorderColor = MaterialTheme.colorScheme.outline,
                            focusedTextColor = MaterialTheme.colorScheme.onSurface,
                            unfocusedTextColor = MaterialTheme.colorScheme.onSurface,
                        ),
                    )
                }

                if (reportState is ReportUiState.Error) {
                    Text(
                        reportState.message,
                        style = MaterialTheme.typography.bodySmall,
                        color = LocalStatusColors.current.error,
                    )
                }
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.End,
                ) {
                    TextButton(onClick = { dialogOpen = false; onDismiss() }) {
                        Text("Cancel", color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                    Spacer(modifier = Modifier.width(8.dp))
                    Button(
                        onClick = { onSubmit(message, carrierText.trim()) },
                        enabled = message.isNotBlank() && reportState !is ReportUiState.Sending,
                        colors = ButtonDefaults.buttonColors(
                            containerColor = MaterialTheme.colorScheme.primary,
                            contentColor = MaterialTheme.colorScheme.onPrimary,
                        ),
                    ) {
                        if (reportState is ReportUiState.Sending) {
                            CircularProgressIndicator(
                                modifier = Modifier.size(18.dp),
                                strokeWidth = 2.dp,
                                color = MaterialTheme.colorScheme.onPrimary,
                            )
                        } else {
                            Text("Send")
                        }
                    }
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
