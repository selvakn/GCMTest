package com.tw.pushlatency.enrollment

import android.content.Context
import android.util.Log
import com.tw.pushlatency.network.CoordinatorApi

private const val TAG = "EnrollmentManager"
private const val PLATFORM = "android"

/**
 * Enrolls this device with the coordinator on first launch (FR-017), skips
 * re-enrollment when already active (FR-018), and re-enrolls (same
 * install_id, new token) whenever the platform issues a new push identity
 * (FR-019). Enrollment failures are logged and surfaced non-blockingly by
 * the caller (FR-020) — they never crash or block the app.
 */
class EnrollmentManager(private val context: Context) {

    private val installIdProvider = InstallIdProvider(context)
    private val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    sealed class Result {
        data object AlreadyEnrolled : Result()
        data object Enrolled : Result()
        data class Failed(val cause: Throwable) : Result()
    }

    /** Called on app launch: enrolls only if there is no valid local record (FR-017, FR-018). */
    fun enrollIfNeeded(pushToken: String): Result {
        if (prefs.getBoolean(KEY_ENROLLED, false)) {
            return Result.AlreadyEnrolled
        }
        return enrollOrUpdate(pushToken)
    }

    /** Called whenever the current push token changes (FR-019), including first enrollment. */
    fun enrollOrUpdate(pushToken: String): Result {
        val installId = installIdProvider.installId()
        return try {
            CoordinatorApi.enrollDevice(installId, PLATFORM, pushToken)
            prefs.edit().putBoolean(KEY_ENROLLED, true).apply()
            Result.Enrolled
        } catch (e: Exception) {
            Log.w(TAG, "enrollment failed, will not block app usage", e)
            Result.Failed(e)
        }
    }

    companion object {
        private const val PREFS_NAME = "push_latency_enrollment"
        private const val KEY_ENROLLED = "enrolled"
    }
}
