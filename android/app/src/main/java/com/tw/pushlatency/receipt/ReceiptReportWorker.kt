package com.tw.pushlatency.receipt

import android.content.Context
import androidx.work.BackoffPolicy
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.Data
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkRequest
import androidx.work.WorkerParameters
import com.tw.pushlatency.enrollment.InstallIdProvider
import com.tw.pushlatency.network.CoordinatorApi
import java.time.Instant
import java.util.concurrent.TimeUnit

/**
 * Reports receipt of a round to the coordinator, retrying with exponential
 * backoff on failure (FR-023) instead of silently dropping the report — a
 * lost report is otherwise indistinguishable from non-delivery in the
 * fleet-wide statistics this tool exists to produce.
 */
class ReceiptReportWorker(
    context: Context,
    params: WorkerParameters,
) : CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        val roundId = inputData.getString(KEY_ROUND_ID) ?: return Result.failure()
        val receivedAtMillis = inputData.getLong(KEY_RECEIVED_AT_MILLIS, -1L)
        if (receivedAtMillis < 0L) return Result.failure()

        val installId = InstallIdProvider(applicationContext).installId()

        return try {
            CoordinatorApi.reportReceipt(roundId, installId, Instant.ofEpochMilli(receivedAtMillis))
            Result.success()
        } catch (e: Exception) {
            Result.retry()
        }
    }

    companion object {
        private const val KEY_ROUND_ID = "round_id"
        private const val KEY_RECEIVED_AT_MILLIS = "received_at_millis"

        fun enqueue(context: Context, roundId: String, receivedAt: Instant) {
            val data = Data.Builder()
                .putString(KEY_ROUND_ID, roundId)
                .putLong(KEY_RECEIVED_AT_MILLIS, receivedAt.toEpochMilli())
                .build()

            val constraints = Constraints.Builder()
                .setRequiredNetworkType(NetworkType.CONNECTED)
                .build()

            val request = OneTimeWorkRequestBuilder<ReceiptReportWorker>()
                .setInputData(data)
                .setConstraints(constraints)
                .setBackoffCriteria(
                    BackoffPolicy.EXPONENTIAL,
                    WorkRequest.MIN_BACKOFF_MILLIS,
                    TimeUnit.MILLISECONDS,
                )
                .build()

            WorkManager.getInstance(context).enqueue(request)
        }
    }
}
