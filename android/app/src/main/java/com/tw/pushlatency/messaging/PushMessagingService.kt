package com.tw.pushlatency.messaging

import android.util.Log
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import com.tw.pushlatency.PushLatencyApp
import com.tw.pushlatency.data.MessageEntity
import com.tw.pushlatency.enrollment.EnrollmentManager
import com.tw.pushlatency.receipt.ReceiptReportWorker
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import java.time.Instant

private const val TAG = "PushMessagingService"

/**
 * Receives round notifications in the background (FR-021) and reports
 * receipt back to the coordinator, retrying on failure via WorkManager
 * (FR-023) rather than risking a silently dropped report.
 */
class PushMessagingService : FirebaseMessagingService() {

    private val scope = CoroutineScope(Dispatchers.IO)

    override fun onMessageReceived(message: RemoteMessage) {
        // Capture arrival time as precisely as possible, before any further
        // processing (FR-021) — this is the timestamp used for latency.
        val receivedAt = Instant.now()

        val roundId = message.data["round_id"] ?: run {
            Log.w(TAG, "received message with no round_id, ignoring")
            return
        }
        val sentAtRaw = message.data["sent_at"] ?: run {
            Log.w(TAG, "received message with no sent_at, ignoring")
            return
        }
        val sentAt = runCatching { Instant.parse(sentAtRaw) }.getOrElse {
            Log.w(TAG, "malformed sent_at '$sentAtRaw', ignoring")
            return
        }

        val latencyMillis = receivedAt.toEpochMilli() - sentAt.toEpochMilli()

        scope.launch {
            val app = application as PushLatencyApp
            app.database.messageDao().insert(
                MessageEntity(
                    roundId = roundId,
                    sentAtMillis = sentAt.toEpochMilli(),
                    receivedAtMillis = receivedAt.toEpochMilli(),
                    latencyMillis = latencyMillis,
                )
            )
        }

        ReceiptReportWorker.enqueue(applicationContext, roundId, receivedAt)
    }

    override fun onNewToken(token: String) {
        // The platform issued a new push identity; re-enroll with it so the
        // fleet roster stays accurate, keeping the same install_id (FR-019).
        EnrollmentManager(applicationContext).enrollOrUpdate(pushToken = token)
    }
}
