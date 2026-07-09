package com.tw.pushlatency

import android.app.Application
import android.content.Intent
import androidx.room.Room
import com.tw.pushlatency.data.AppDatabase
import com.tw.pushlatency.experiment.AlwaysOnBackgroundService

class PushLatencyApp : Application() {

    lateinit var database: AppDatabase
        private set

    override fun onCreate() {
        super.onCreate()
        database = Room.databaseBuilder(this, AppDatabase::class.java, "push-latency.db").build()

        // Diagnostic-only (see docs/android-background-execution-limits.md):
        // starts the "always running" background service the moment the app
        // process starts, so it pre-exists independently of any push.
        startService(Intent(this, AlwaysOnBackgroundService::class.java))
    }
}
