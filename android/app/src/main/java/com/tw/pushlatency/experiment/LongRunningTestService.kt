package com.tw.pushlatency.experiment

/** Uses the `specialUse` foreground service type (see AndroidManifest.xml). */
class LongRunningTestService : BaseExperimentService() {
    override val label = "specialUse"
}
