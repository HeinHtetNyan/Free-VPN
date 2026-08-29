package com.syvpn.app.ads

import android.annotation.SuppressLint
import android.graphics.Color as AndroidColor
import android.webkit.WebView
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.height
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import com.syvpn.app.BuildConfig
import com.syvpn.app.ui.theme.DarkBackground

/**
 * Adsterra has no native Android SDK (see docs/MONETIZATION.md) — ads are
 * served via a WebView loading their ad tag. Deliberately scoped to a
 * contained, fixed-size banner: per docs/PLAY_STORE_COMPLIANCE.md, Popunder
 * and Social Bar formats are NOT to be used in this app (Play Store
 * rejection risk) — only Banner/Native/Interstitial, rendered inline like
 * this, never as a full-screen takeover the user didn't tap into.
 *
 * Zone ID comes from BuildConfig.ADSTERRA_BANNER_ZONE_ID, sourced from
 * android/local.properties (gitignored, real credentials never committed —
 * see android/local.properties.example). Falls back to a placeholder if
 * that file/env var is missing. Ad tag markup below is the real snippet
 * from the Adsterra dashboard for the 320x50 Banner unit.
 */
@SuppressLint("SetJavaScriptEnabled")
@Composable
fun AdsterraBannerAd(modifier: Modifier = Modifier) {
    AndroidView(
        modifier = modifier.height(50.dp).background(DarkBackground),
        factory = { context ->
            WebView(context).apply {
                // WebView paints opaque white by default — without this, the
                // banner is a stark white bar against the app's dark theme
                // for however long the ad script takes to load (or if it
                // fails to load at all).
                setBackgroundColor(AndroidColor.TRANSPARENT)
                // A hardware-accelerated WebView keeps its own compositor
                // layer alongside Compose's — when the ad creative loads/
                // resizes, that layer can resync and blank the *entire*
                // window for a frame or two (well-documented WebView
                // quirk, especially on MIUI). Software rendering avoids the
                // separate layer entirely, at the cost of slightly slower
                // ad rendering — an easy trade for a 50dp banner.
                setLayerType(android.view.View.LAYER_TYPE_SOFTWARE, null)
                settings.javaScriptEnabled = true
                settings.domStorageEnabled = true
                loadDataWithBaseURL(
                    "https://www.adsterra.com",
                    adTagHtml(zoneId = BuildConfig.ADSTERRA_BANNER_ZONE_ID),
                    "text/html",
                    "UTF-8",
                    null,
                )
            }
        },
    )
}

private fun adTagHtml(zoneId: String): String = """
    <html><body style="margin:0;padding:0;background:#0A0F1C;">
    <script>
    atOptions = {
    'key' : '$zoneId',
    'format' : 'iframe',
    'height' : 50,
    'width' : 320,
    'params' : {}
    };
    </script>
    <script src="https://www.highrevenueformat.com/$zoneId/invoke.js"></script>
    </body></html>
""".trimIndent()
