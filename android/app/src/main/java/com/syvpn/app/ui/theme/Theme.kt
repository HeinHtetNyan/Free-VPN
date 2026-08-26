package com.syvpn.app.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color

/** Per-connection-state colors — outside Material3's fixed color roles (there's
 * no built-in "connecting" or "idle" slot), so exposed as our own small
 * CompositionLocal alongside MaterialTheme rather than overloading `error`
 * for anything that isn't actually an error. */
data class StatusColors(
    val idle: Color,
    val connecting: Color,
    val connected: Color,
    val error: Color,
)

val LocalStatusColors = staticCompositionLocalOf {
    StatusColors(
        idle = StateIdleDark,
        connecting = StateConnectingDark,
        connected = StateConnectedDark,
        error = StateErrorDark,
    )
}

private val DarkScheme = darkColorScheme(
    primary = AccentTealDark,
    onPrimary = Color(0xFF00281C),
    background = DarkBackground,
    onBackground = DarkTextPrimary,
    surface = DarkSurface,
    onSurface = DarkTextPrimary,
    surfaceVariant = DarkSurfaceAlt,
    onSurfaceVariant = DarkTextSecondary,
    outline = DarkOutline,
    error = StateErrorDark,
    onError = Color(0xFF3A1206),
)

private val LightScheme = lightColorScheme(
    primary = AccentTealLight,
    onPrimary = Color.White,
    background = LightBackground,
    onBackground = LightTextPrimary,
    surface = LightSurface,
    onSurface = LightTextPrimary,
    surfaceVariant = LightSurfaceAlt,
    onSurfaceVariant = LightTextSecondary,
    outline = LightOutline,
    error = StateErrorLight,
    onError = Color.White,
)

@Composable
fun VpnAppTheme(content: @Composable () -> Unit) {
    val dark = isSystemInDarkTheme()
    val colorScheme = if (dark) DarkScheme else LightScheme
    val statusColors = if (dark) {
        StatusColors(StateIdleDark, StateConnectingDark, StateConnectedDark, StateErrorDark)
    } else {
        StatusColors(StateIdleLight, StateConnectingLight, StateConnectedLight, StateErrorLight)
    }
    CompositionLocalProvider(LocalStatusColors provides statusColors) {
        MaterialTheme(colorScheme = colorScheme, typography = VpnTypography, content = content)
    }
}
