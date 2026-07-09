package com.tw.pushlatency.experiment

/**
 * Uses the `dataSync` foreground service type (see AndroidManifest.xml) —
 * unlike `specialUse`, Android 15 imposes a multi-hour timeout on this type
 * before the system stops it, which this 5-minute experiment is too short
 * to observe; it's here to confirm dataSync behaves the same as specialUse
 * within that window.
 */
class DataSyncTestService : BaseExperimentService() {
    override val label = "dataSync"
}
