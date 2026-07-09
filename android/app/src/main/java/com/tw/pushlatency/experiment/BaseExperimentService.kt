package com.tw.pushlatency.experiment

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Intent
import android.os.Build
import android.os.IBinder
import android.util.Log
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch

private const val TAG = "SvcExperiment"
private const val NOTIFICATION_CHANNEL_ID = "svc_experiment"
private const val NOTIFICATION_ID = 42

/**
 * Diagnostic-only base service (not part of the product) used to empirically
 * measure how long Android lets FCM-triggered background work keep running
 * before the OS intervenes, across app-foreground/background/Doze states,
 * foreground-service types, and idle-timer-vs-active-socket workloads.
 * See docs/android-background-execution-limits.md.
 */
abstract class BaseExperimentService : Service() {

    /** A short label identifying this service class in log lines, e.g. "specialUse", "dataSync". */
    abstract val label: String

    private val scope = CoroutineScope(Dispatchers.IO)
    private var job: Job? = null

    override fun onCreate() {
        super.onCreate()
        Log.i(TAG, "[$label] onCreate pid=${android.os.Process.myPid()}")
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val asForeground = intent?.getBooleanExtra(EXTRA_FOREGROUND, false) ?: false
        val useSocket = intent?.getBooleanExtra(EXTRA_USE_SOCKET, false) ?: false
        Log.i(TAG, "[$label] onStartCommand asForeground=$asForeground useSocket=$useSocket startedAtMillis=${System.currentTimeMillis()}")

        if (asForeground) {
            startForeground(NOTIFICATION_ID, buildNotification())
        }

        job?.cancel()
        job = scope.launch {
            if (useSocket) {
                ExperimentRunner.runSocketLoop(label)
            } else {
                ExperimentRunner.runTimerLoop(label)
            }
            stopSelf(startId)
        }

        return START_NOT_STICKY
    }

    override fun onDestroy() {
        Log.i(TAG, "[$label] onDestroy called — service is being torn down")
        job?.cancel()
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private fun buildNotification(): Notification {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                NOTIFICATION_CHANNEL_ID,
                "Service experiment",
                NotificationManager.IMPORTANCE_LOW,
            )
            getSystemService(NotificationManager::class.java).createNotificationChannel(channel)
        }
        return Notification.Builder(this, NOTIFICATION_CHANNEL_ID)
            .setContentTitle("Long-running service experiment ($label)")
            .setContentText("Measuring background execution limits")
            .setSmallIcon(android.R.drawable.ic_popup_sync)
            .build()
    }

    companion object {
        const val EXTRA_FOREGROUND = "as_foreground"
        const val EXTRA_USE_SOCKET = "use_socket"
    }
}
