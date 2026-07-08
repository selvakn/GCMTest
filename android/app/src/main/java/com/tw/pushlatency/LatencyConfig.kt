package com.tw.pushlatency

import android.content.Context

enum class LatencyCategory { GOOD, SLOW, ANOMALOUS }

/**
 * The "slow" threshold is a runtime SharedPreferences value, not a compiled
 * constant (FR-027) — it can be changed (e.g. via `adb shell` during QA)
 * without rebuilding the app. [DEFAULT_THRESHOLD_MILLIS] mirrors the
 * original system's 2-second default (spec.md Assumptions).
 */
object LatencyConfig {
    private const val PREFS_NAME = "push_latency_config"
    private const val KEY_SLOW_THRESHOLD_MILLIS = "slow_threshold_millis"
    const val DEFAULT_THRESHOLD_MILLIS = 2000L

    fun slowThresholdMillis(context: Context): Long {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        return prefs.getLong(KEY_SLOW_THRESHOLD_MILLIS, DEFAULT_THRESHOLD_MILLIS)
    }

    fun categorize(latencyMillis: Long, thresholdMillis: Long): LatencyCategory = when {
        latencyMillis <= 0 -> LatencyCategory.ANOMALOUS
        latencyMillis <= thresholdMillis -> LatencyCategory.GOOD
        else -> LatencyCategory.SLOW
    }
}
