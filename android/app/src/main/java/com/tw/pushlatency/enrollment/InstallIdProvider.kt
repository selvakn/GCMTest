package com.tw.pushlatency.enrollment

import android.content.Context
import java.util.UUID

/**
 * Generates a stable, app-generated installation ID once and persists it
 * locally (Clarification Q1, spec.md) — this is the device's primary
 * identity, kept stable across push-token rotation. It does not survive a
 * full uninstall + local-data wipe (research.md §4, known/accepted limitation).
 */
class InstallIdProvider(context: Context) {

    private val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    fun installId(): String {
        prefs.getString(KEY_INSTALL_ID, null)?.let { return it }

        val generated = UUID.randomUUID().toString()
        prefs.edit().putString(KEY_INSTALL_ID, generated).apply()
        return generated
    }

    companion object {
        private const val PREFS_NAME = "push_latency_enrollment"
        private const val KEY_INSTALL_ID = "install_id"
    }
}
