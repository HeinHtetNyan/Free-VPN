package com.syvpn.app.ads

import android.annotation.SuppressLint
import android.webkit.WebView
import androidx.compose.foundation.layout.height
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView

/**
 * Adsterra has no native Android SDK (see docs/MONETIZATION.md) — ads are
 * served via a WebView loading their ad tag. Deliberately scoped to a
 * contained, fixed-size banner: per docs/PLAY_STORE_COMPLIANCE.md, Popunder
 * and Social Bar formats are NOT to be used in this app (Play Store
 * rejection risk) — only Banner/Native/Interstitial, rendered inline like
 * this, never as a full-screen takeover the user didn't tap into.
 *
 * TODO: replace ADSTERRA_ZONE_ID_PLACEHOLDER with a real zone ID once an
 * Adsterra publisher account exists (docs/OPEN_QUESTIONS.md). Ad tag markup
 * below is a placeholder shape, not copied from Adsterra's real docs —
 * confirm the exact script snippet in the Adsterra dashboard before shipping.
 */
@SuppressLint("SetJavaScriptEnabled")
@Composable
fun AdsterraBannerAd(modifier: Modifier = Modifier) {
    AndroidView(
        modifier = modifier.height(50.dp),
        factory = { context ->
            WebView(context).apply {
                settings.javaScriptEnabled = true
                loadDataWithBaseURL(
                    "https://www.adsterra.com",
                    adTagHtml(zoneId = "ADSTERRA_ZONE_ID_PLACEHOLDER"),
                    "text/html",
                    "UTF-8",
                    null,
                )
            }
        },
    )
}

private fun adTagHtml(zoneId: String): String = """
    <html><body style="margin:0;padding:0;">
    <!-- TODO: replace with Adsterra's actual Banner ad tag for zoneId=$zoneId
         from the publisher dashboard once the account exists. -->
    <div id="ad-container"></div>
    </body></html>
""".trimIndent()
