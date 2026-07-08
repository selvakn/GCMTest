plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
    id("com.google.gms.google-services")
    id("com.google.devtools.ksp")
}

android {
    namespace = "com.tw.pushlatency"
    compileSdk = 34

    defaultConfig {
        applicationId = "com.tw.pushlatency"
        minSdk = 26
        targetSdk = 34
        versionCode = 1
        versionName = "1.0"
    }

    // Per-build/environment configuration (FR-028): the coordinator address and the
    // Firebase project (via each flavor's own google-services.json under
    // src/<flavor>/google-services.json) are both configurable per build, not fixed in code.
    flavorDimensions += "environment"
    productFlavors {
        create("dev") {
            dimension = "environment"
            applicationIdSuffix = ".dev"
            buildConfigField("String", "COORDINATOR_BASE_URL", "\"http://10.0.2.2:8080/v1\"")
        }
        create("staging") {
            dimension = "environment"
            applicationIdSuffix = ".staging"
            buildConfigField("String", "COORDINATOR_BASE_URL", "\"https://staging-coordinator.example.com/v1\"")
        }
        create("prod") {
            dimension = "environment"
            buildConfigField("String", "COORDINATOR_BASE_URL", "\"https://coordinator.example.com/v1\"")
        }
        create("docker") {
            dimension = "environment"
            applicationIdSuffix = ".docker"
            // Matches the `coordinator` service name in docker-compose.yml — for
            // running the client inside the docker-android emulator alongside the
            // coordinator container, where the 10.0.2.2 host-loopback trick used by
            // `dev` doesn't reach a separate container.
            buildConfigField("String", "COORDINATOR_BASE_URL", "\"http://coordinator:8080/v1\"")
        }
        create("device") {
            dimension = "environment"
            applicationIdSuffix = ".device"
            // For a real device connected over USB with `adb reverse tcp:8080 tcp:<host-port>`
            // forwarding the device's own localhost:8080 back to the coordinator, regardless
            // of the device's own Wi-Fi/network configuration.
            buildConfigField("String", "COORDINATOR_BASE_URL", "\"http://localhost:8080/v1\"")
        }
    }

    // Release signing comes from environment variables so the keystore itself
    // is never committed (FR-030) — CI decodes a secret into this path and
    // supplies the passwords; a local unsigned release build is still
    // possible (signingConfig only applied when the keystore file exists).
    val releaseKeystorePath = System.getenv("RELEASE_KEYSTORE_PATH")
    signingConfigs {
        if (releaseKeystorePath != null) {
            create("release") {
                storeFile = file(releaseKeystorePath)
                storePassword = System.getenv("RELEASE_KEYSTORE_PASSWORD")
                keyAlias = System.getenv("RELEASE_KEY_ALIAS")
                keyPassword = System.getenv("RELEASE_KEY_PASSWORD")
            }
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            if (releaseKeystorePath != null) {
                signingConfig = signingConfigs.getByName("release")
            }
        }
    }

    buildFeatures {
        buildConfig = true
        viewBinding = true
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }
}

dependencies {
    implementation("androidx.core:core-ktx:1.13.1")
    implementation("androidx.appcompat:appcompat:1.7.0")
    implementation("com.google.android.material:material:1.12.0")
    implementation("androidx.recyclerview:recyclerview:1.3.2")
    implementation("androidx.constraintlayout:constraintlayout:2.1.4")

    implementation(platform("com.google.firebase:firebase-bom:33.1.2"))
    implementation("com.google.firebase:firebase-messaging-ktx")

    implementation("androidx.work:work-runtime-ktx:2.9.1")

    implementation("androidx.room:room-runtime:2.6.1")
    implementation("androidx.room:room-ktx:2.6.1")
    ksp("androidx.room:room-compiler:2.6.1")

    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.8.1")
}
