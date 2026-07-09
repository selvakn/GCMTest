package com.tw.pushlatency.experiment

import kotlinx.coroutines.flow.MutableSharedFlow

/**
 * In-process pub/sub used to hand a push-triggered event to an already-running
 * background component, without starting/binding a new service — simulates
 * "the app has a persistent background service; push notifications are just
 * routed to it," as opposed to a service freshly started by the push itself
 * (see [BaseExperimentService], which tests that other case).
 */
object ExperimentEventBus {
    val events = MutableSharedFlow<String>(extraBufferCapacity = 8)
}
