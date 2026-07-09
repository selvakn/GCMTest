package com.tw.pushlatency.messaging

import android.content.Intent
import android.os.Build
import android.util.Log
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import com.tw.pushlatency.PushLatencyApp
import com.tw.pushlatency.data.MessageEntity
import com.tw.pushlatency.enrollment.EnrollmentManager
import com.tw.pushlatency.experiment.BaseExperimentService
import com.tw.pushlatency.experiment.DataSyncTestService
import com.tw.pushlatency.experiment.ExperimentEventBus
import com.tw.pushlatency.experiment.LongRunningTestService
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
        // Diagnostic-only hook (not part of the product): a push carrying an
        // "experiment" data field triggers the long-running-service
        // execution-limit experiment instead of normal round handling.
        message.data["experiment"]?.let { experiment ->
            handleExperimentTrigger(experiment)
            return
        }

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

    private fun handleExperimentTrigger(experiment: String) {
        Log.i(TAG, "experiment trigger received: '$experiment' at ${Instant.now()}")

        if (experiment == "existing_service_socket") {
            // Unlike the other experiments, this does NOT start a new
            // service — it only hands an event to whatever is already
            // listening on the event bus (AlwaysOnBackgroundService, started
            // once when the app process started). No startService() call
            // happens here at all.
            scope.launch {
                ExperimentEventBus.events.emit("socket_connect")
                Log.i(TAG, "emitted event to ExperimentEventBus for '$experiment'")
            }
            return
        }

        val (serviceClass, foreground, useSocket) = when (experiment) {
            "bg_service" -> Triple(LongRunningTestService::class.java, false, false)
            "fg_service" -> Triple(LongRunningTestService::class.java, true, false)
            "bg_service_socket" -> Triple(LongRunningTestService::class.java, false, true)
            "fg_service_socket" -> Triple(LongRunningTestService::class.java, true, true)
            // dataSync only means something once promoted to foreground — a
            // non-promoted service has no foreground-service type at all.
            "datasync_service" -> Triple(DataSyncTestService::class.java, true, false)
            else -> {
                Log.w(TAG, "unknown experiment type '$experiment', ignoring")
                return
            }
        }

        val intent = Intent(this, serviceClass).apply {
            putExtra(BaseExperimentService.EXTRA_FOREGROUND, foreground)
            putExtra(BaseExperimentService.EXTRA_USE_SOCKET, useSocket)
        }
        try {
            if (foreground && Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                startForegroundService(intent)
            } else {
                startService(intent)
            }
            Log.i(TAG, "service start call for '$experiment' returned normally")
        } catch (e: Exception) {
            Log.e(TAG, "service start call for '$experiment' threw ${e.javaClass.simpleName}: ${e.message}")
        }
    }

    override fun onNewToken(token: String) {
        // The platform issued a new push identity; re-enroll with it so the
        // fleet roster stays accurate, keeping the same install_id (FR-019).
        // Enrollment does blocking network I/O — never on the main thread.
        scope.launch {
            EnrollmentManager(applicationContext).enrollOrUpdate(pushToken = token)
        }
    }
}
