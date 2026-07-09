package com.tw.pushlatency.experiment

import android.util.Log
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import java.io.BufferedReader
import java.io.InputStreamReader
import java.io.PrintWriter
import java.net.Socket
import kotlin.coroutines.coroutineContext

private const val TAG = "SvcExperiment"
const val MAX_DURATION_SECONDS = 300L
private const val HEARTBEAT_INTERVAL_SECONDS = 10L
private const val MOCK_SOCKET_HOST = "localhost"
private const val MOCK_SOCKET_PORT = 9090

/** Shared heartbeat loop used by both diagnostic test services. */
object ExperimentRunner {

    /** Plain timer heartbeat: no I/O, just proves the process/thread is alive. */
    suspend fun runTimerLoop(label: String) {
        var elapsed = 0L
        while (coroutineContext.isActive && elapsed < MAX_DURATION_SECONDS) {
            Log.i(TAG, "[$label] heartbeat elapsedSeconds=$elapsed")
            delay(HEARTBEAT_INTERVAL_SECONDS * 1000)
            elapsed += HEARTBEAT_INTERVAL_SECONDS
        }
        Log.i(TAG, "[$label] completed full ${MAX_DURATION_SECONDS}s without being killed")
    }

    /**
     * Holds one long-lived TCP socket open for the whole run, sending a ping
     * line every [HEARTBEAT_INTERVAL_SECONDS] and reading back the reply —
     * simulating a websocket-style persistent connection (e.g. a chat app's
     * realtime channel) rather than an idle timer, to see whether active
     * network I/O changes how long the OS lets the service keep running.
     */
    suspend fun runSocketLoop(label: String) {
        var elapsed = 0L
        try {
            Socket(MOCK_SOCKET_HOST, MOCK_SOCKET_PORT).use { socket ->
                socket.soTimeout = (HEARTBEAT_INTERVAL_SECONDS * 1000 + 5000).toInt()
                val out = PrintWriter(socket.getOutputStream(), true)
                val input = BufferedReader(InputStreamReader(socket.getInputStream()))
                Log.i(TAG, "[$label] socket connected to $MOCK_SOCKET_HOST:$MOCK_SOCKET_PORT")

                while (coroutineContext.isActive && elapsed < MAX_DURATION_SECONDS) {
                    out.println("ping seq=${elapsed / HEARTBEAT_INTERVAL_SECONDS} elapsedSeconds=$elapsed")
                    val reply = input.readLine()
                    Log.i(TAG, "[$label] socket heartbeat elapsedSeconds=$elapsed reply=$reply")
                    delay(HEARTBEAT_INTERVAL_SECONDS * 1000)
                    elapsed += HEARTBEAT_INTERVAL_SECONDS
                }
                Log.i(TAG, "[$label] completed full ${MAX_DURATION_SECONDS}s with socket still open")
            }
        } catch (e: Exception) {
            Log.e(TAG, "[$label] socket loop failed at elapsedSeconds=$elapsed: ${e.javaClass.simpleName}: ${e.message}")
        }
    }
}
