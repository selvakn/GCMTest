package com.tw.pushlatency.network

import com.tw.pushlatency.BuildConfig
import java.io.OutputStreamWriter
import java.net.HttpURLConnection
import java.net.URL
import java.time.Instant
import org.json.JSONObject

/**
 * Minimal HTTP client for the coordinator API (contracts/openapi.yaml). The
 * coordinator base URL is per-build configuration (FR-028), read from
 * BuildConfig so it varies by flavor without a rebuild of the source itself.
 */
object CoordinatorApi {

    private val baseUrl: String get() = BuildConfig.COORDINATOR_BASE_URL

    /** POST /v1/devices — enroll or re-enroll. Throws on any non-2xx response. */
    fun enrollDevice(installId: String, platform: String, pushToken: String) {
        val body = JSONObject()
            .put("install_id", installId)
            .put("platform", platform)
            .put("push_token", pushToken)
        postJson("$baseUrl/devices", body)
    }

    /** POST /v1/rounds/{roundId}/receipts — report that this device received a round. */
    fun reportReceipt(roundId: String, installId: String, receivedAt: Instant) {
        val body = JSONObject()
            .put("install_id", installId)
            .put("received_at", receivedAt.toString())
        postJson("$baseUrl/rounds/$roundId/receipts", body)
    }

    private fun postJson(url: String, body: JSONObject) {
        val connection = URL(url).openConnection() as HttpURLConnection
        try {
            connection.requestMethod = "POST"
            connection.doOutput = true
            connection.setRequestProperty("Content-Type", "application/json")
            connection.connectTimeout = 10_000
            connection.readTimeout = 10_000

            OutputStreamWriter(connection.outputStream).use { it.write(body.toString()) }

            val status = connection.responseCode
            if (status !in 200..299) {
                throw CoordinatorApiException(status)
            }
        } finally {
            connection.disconnect()
        }
    }
}

class CoordinatorApiException(val statusCode: Int) :
    Exception("Coordinator API call failed with status $statusCode")
