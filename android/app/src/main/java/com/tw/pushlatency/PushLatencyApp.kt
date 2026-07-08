package com.tw.pushlatency

import android.app.Application
import androidx.room.Room
import com.tw.pushlatency.data.AppDatabase

class PushLatencyApp : Application() {

    lateinit var database: AppDatabase
        private set

    override fun onCreate() {
        super.onCreate()
        database = Room.databaseBuilder(this, AppDatabase::class.java, "push-latency.db").build()
    }
}
