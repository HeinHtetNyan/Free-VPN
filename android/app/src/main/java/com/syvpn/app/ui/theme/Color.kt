package com.syvpn.app.ui.theme

import androidx.compose.ui.graphics.Color

// "Night signal" palette — deep indigo-black grounds (not flat near-black),
// a jade-teal brand accent, and one distinct color per connection state so
// the status card reads at a glance without depending on the label text.

// Dark (primary design target — this is a connect/disconnect utility most
// often glanced at, and a dark ground suits "signal in the dark" better
// than a bright one).
val DarkBackground = Color(0xFF0A0F1C)
val DarkSurface = Color(0xFF131C30)
val DarkSurfaceAlt = Color(0xFF1C2740)
val DarkTextPrimary = Color(0xFFEDF1F7)
val DarkTextSecondary = Color(0xFF92A0BF)
val DarkOutline = Color(0xFF2A3550)

// Light
val LightBackground = Color(0xFFF3F5FA)
val LightSurface = Color(0xFFFFFFFF)
val LightSurfaceAlt = Color(0xFFE9EDF6)
val LightTextPrimary = Color(0xFF10182A)
val LightTextSecondary = Color(0xFF525F7D)
val LightOutline = Color(0xFFD6DCEA)

// Brand accent (buttons, selection, focus) — stable across both themes and
// independent of connection state, so the primary action never changes
// color based on what it's about to do.
val AccentTealDark = Color(0xFF35D0A0)
val AccentTealLight = Color(0xFF0E9A73)

// Per-state colors for the status card. Idle/Connecting/Error read as
// distinct hues, not shades of the same alert color.
val StateIdleDark = Color(0xFF6B7A9C)
val StateIdleLight = Color(0xFF64719A)
val StateConnectingDark = Color(0xFFF2B84B)
val StateConnectingLight = Color(0xFFB9791A)
val StateConnectedDark = AccentTealDark
val StateConnectedLight = AccentTealLight
val StateErrorDark = Color(0xFFF0754A)
val StateErrorLight = Color(0xFFC94E27)
