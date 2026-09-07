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
 * rejection risk) — only plain Banner (this composable, proven safe by live
 * testing), rendered inline, never as a full-screen takeover. Native Banner
 * and Smartlink are explicitly NOT used for anything beyond this file's
 * fixed-size iframe technique — see ads/ConnectInterstitialAd.kt's doc
 * comment for why (a Native Banner zone hijacked arbitrary taps into
 * external redirects when tried as a full-screen interstitial).
 *
 * zoneId/widthDp/heightDp default to the original bottom banner (320x50,
 * BuildConfig.ADSTERRA_BANNER_ZONE_ID) so existing call sites don't need to
 * change; pass a different zone/size for another placement (e.g. the
 * content banner between the location list and "Report an issue").
 */
@SuppressLint("SetJavaScriptEnabled")
@Composable
fun AdsterraBannerAd(
    modifier: Modifier = Modifier,
    zoneId: String = BuildConfig.ADSTERRA_BANNER_ZONE_ID,
    widthDp: Int = 320,
    heightDp: Int = 50,
) {
    AndroidView(
        modifier = modifier.height(heightDp.dp).background(DarkBackground),
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
                // ad rendering — an easy trade for a small banner.
                setLayerType(android.view.View.LAYER_TYPE_SOFTWARE, null)
                settings.javaScriptEnabled = true
                settings.domStorageEnabled = true
                loadDataWithBaseURL(
                    "https://www.adsterra.com",
                    adTagHtml(zoneId = zoneId, widthPx = widthDp, heightPx = heightDp),
                    "text/html",
                    "UTF-8",
                    null,
                )
            }
        },
    )
}

private fun adTagHtml(zoneId: String, widthPx: Int, heightPx: Int): String = """
    <html><body style="margin:0;padding:0;background:#0A0F1C;">
    <script>
    atOptions = {
    'key' : '$zoneId',
    'format' : 'iframe',
    'height' : $heightPx,
    'width' : $widthPx,
    'params' : {}
    };
    </script>
    <script src="https://www.highrevenueformat.com/$zoneId/invoke.js"></script>
    </body></html>
""".trimIndent()
