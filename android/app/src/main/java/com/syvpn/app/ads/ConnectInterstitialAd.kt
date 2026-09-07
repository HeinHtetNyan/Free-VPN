package com.syvpn.app.ads

import android.annotation.SuppressLint
import android.graphics.Color as AndroidColor
import android.webkit.WebView
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.ButtonDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import com.syvpn.app.BuildConfig
import com.syvpn.app.ui.theme.DarkBackground
import kotlinx.coroutines.delay

/**
 * NOT CURRENTLY WIRED UP — see MainActivity.onConnectClick()'s doc comment.
 *
 * Full-screen Adsterra Native Banner interstitial, meant to show right after
 * the user taps Connect. Renders fine technically (a Native Banner is just
 * an async script + container div, same embedding technique as
 * AdsterraBannerAd's bottom banner — unlike a Smartlink, which needs a real
 * browser tab because its redirect chain fires an intent-based hop WebView
 * blocks without a user gesture).
 *
 * The problem is the ad creative itself, tested live on-device: this zone's
 * inventory used a full-screen invisible overlay that hijacked ANY tap on
 * the WebView into an external browser redirect — confirmed via logcat, an
 * ACTION_VIEW intent to Chrome fired carrying our own zone key
 * (e40eafb272f42d1c0016361c2ca31cbf) with no real ad click involved. That's
 * abusive ad behavior (and a Play Store disruptive-ads risk), not a bug in
 * this file. Left here disconnected in case a better-behaved ad
 * unit/network is worth trying later with the same skip/continue plumbing.
 *
 * The skip/continue control is drawn on an opaque background chip rather
 * than plain colored text — a lesson from testing: the ad creative's own
 * background can be any color, and white text with no backing surface was
 * genuinely invisible against a white creative.
 */
@SuppressLint("SetJavaScriptEnabled")
@Composable
fun ConnectInterstitialAd(onDismiss: () -> Unit) {
    val scriptUrl = BuildConfig.ADSTERRA_CONNECT_NATIVE_BANNER_SCRIPT_URL
    val containerId = BuildConfig.ADSTERRA_CONNECT_NATIVE_BANNER_CONTAINER_ID
    if (scriptUrl.isBlank() || scriptUrl == "ADSTERRA_ZONE_ID_PLACEHOLDER") {
        LaunchedEffect(Unit) { onDismiss() }
        return
    }

    var secondsRemaining by remember { mutableIntStateOf(SKIP_DELAY_SECONDS) }
    LaunchedEffect(Unit) {
        while (secondsRemaining > 0) {
            delay(1_000)
            secondsRemaining -= 1
        }
    }

    Box(modifier = Modifier.fillMaxSize().background(DarkBackground)) {
        AndroidView(
            modifier = Modifier.fillMaxSize(),
            factory = { context ->
                WebView(context).apply {
                    setBackgroundColor(AndroidColor.TRANSPARENT)
                    settings.javaScriptEnabled = true
                    settings.domStorageEnabled = true
                    loadDataWithBaseURL(
                        "https://www.adsterra.com",
                        adTagHtml(scriptUrl = scriptUrl, containerId = containerId),
                        "text/html",
                        "UTF-8",
                        null,
                    )
                }
            },
        )
        val chipModifier = Modifier
            .align(Alignment.TopEnd)
            .padding(16.dp)
            .background(Color.Black.copy(alpha = 0.6f), RoundedCornerShape(20.dp))
        if (secondsRemaining > 0) {
            Text(
                "Skip in ${secondsRemaining}s",
                color = Color.White,
                modifier = chipModifier.padding(horizontal = 16.dp, vertical = 10.dp),
            )
        } else {
            TextButton(
                onClick = onDismiss,
                modifier = chipModifier,
                colors = ButtonDefaults.textButtonColors(contentColor = Color.White),
            ) {
                Text("Continue →")
            }
        }
    }
}

private fun adTagHtml(scriptUrl: String, containerId: String): String = """
    <html><body style="margin:0;padding:0;background:#0A0F1C;">
    <script async="async" data-cfasync="false" src="$scriptUrl"></script>
    <div id="$containerId"></div>
    </body></html>
""".trimIndent()

/** Long enough to register as a real impression, short enough not to feel
 * like a wall between the user and the Connect action they just tapped. */
private const val SKIP_DELAY_SECONDS = 5
