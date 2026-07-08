package com.tw.pushlatency.data

import androidx.room.Entity
import androidx.room.PrimaryKey

/** One row in this device's on-screen log: one round it received (FR-024). */
@Entity(tableName = "messages")
data class MessageEntity(
    @PrimaryKey val roundId: String,
    val sentAtMillis: Long,
    val receivedAtMillis: Long,
    val latencyMillis: Long,
)
