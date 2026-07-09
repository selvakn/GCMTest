package com.tw.pushlatency.experiment

import android.app.Service
import android.content.Intent
import android.os.IBinder
import android.util.Log
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch

private const val TAG = "SvcExperiment"

/**
 * A plain background service started once when the app process starts
 * ([com.tw.pushlatency.PushLatencyApp.onCreate]) and otherwise a genuine
 * no-op — simulates "the app always has a background service running."
 * Only when an event arrives on [ExperimentEventBus] does it do anything:
 * establish and hold the mock socket connection. Distinct from
 * [BaseExperimentService], which is freshly started BY the push itself —
 * here the service pre-exists, and the push is only ever routed to it via
 * the event bus, never a new startService()/startForegroundService() call.
 */
class AlwaysOnBackgroundService : Service() {

    private val scope = CoroutineScope(Dispatchers.IO)
    private var socketJob: Job? = null

    override fun onCreate() {
        super.onCreate()
        Log.i(TAG, "[always-on] onCreate pid=${android.os.Process.myPid()} — idle, waiting for event bus")
        scope.launch {
            ExperimentEventBus.events.collect { event ->
                Log.i(TAG, "[always-on] received event '$event' via event bus, starting socket loop")
                socketJob?.cancel()
                socketJob = launch { ExperimentRunner.runSocketLoop("always-on") }
            }
        }
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        Log.i(TAG, "[always-on] onStartCommand — no-op, just keeps the service alive")
        // START_STICKY (unlike BaseExperimentService's START_NOT_STICKY):
        // this service is meant to represent "always running," so ask the
        // system to recreate it if it's ever killed.
        return START_STICKY
    }

    override fun onDestroy() {
        Log.i(TAG, "[always-on] onDestroy called — service is being torn down")
        socketJob?.cancel()
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null
}
