package com.syvpn.app.ads

import android.app.Activity
import android.content.Context
import android.util.Log
import com.google.android.gms.ads.AdError
import com.google.android.gms.ads.AdRequest
import com.google.android.gms.ads.FullScreenContentCallback
import com.google.android.gms.ads.LoadAdError
import com.google.android.gms.ads.interstitial.InterstitialAd
import com.google.android.gms.ads.interstitial.InterstitialAdLoadCallback
import com.syvpn.app.BuildConfig

/**
 * Replaces the never-shipped Adsterra Native Banner interstitial (see git
 * history — ads/ConnectInterstitialAd.kt) which hijacked taps into an
 * external browser on its full-screen WebView. AdMob's own SDK renders and
 * handles taps on the full-screen surface itself — no WebView, no ad-tag
 * injection — so that failure mode doesn't apply here.
 *
 * An InterstitialAd is single-use: load() must be called again after every
 * show (handled internally below).
 */
class ConnectInterstitialAdManager(context: Context) {
    private val appContext = context.applicationContext
    private var interstitialAd: InterstitialAd? = null

    fun load() {
        if (interstitialAd != null) return
        InterstitialAd.load(
            appContext,
            BuildConfig.ADMOB_CONNECT_INTERSTITIAL_UNIT_ID,
            AdRequest.Builder().build(),
            object : InterstitialAdLoadCallback() {
                override fun onAdLoaded(ad: InterstitialAd) {
                    interstitialAd = ad
                }

                override fun onAdFailedToLoad(error: LoadAdError) {
                    interstitialAd = null
                    Log.w(TAG, "Failed to load Connect interstitial: ${error.message}")
                }
            },
        )
    }

    /**
     * Shows the ad if one is loaded and ready; always calls [onDismissed]
     * exactly once either way. Connecting to the VPN must never be gated on
     * ad availability — a missing/failed ad just means the tap proceeds
     * straight to connecting, same as before this ad unit existed.
     */
    fun showIfReady(activity: Activity, onDismissed: () -> Unit) {
        val ad = interstitialAd
        if (ad == null) {
            onDismissed()
            return
        }
        ad.fullScreenContentCallback = object : FullScreenContentCallback() {
            override fun onAdDismissedFullScreenContent() {
                interstitialAd = null
                load()
                onDismissed()
            }

            override fun onAdFailedToShowFullScreenContent(error: AdError) {
                interstitialAd = null
                load()
                onDismissed()
            }
        }
        ad.show(activity)
    }

    private companion object {
        const val TAG = "ConnectInterstitialAd"
    }
}
