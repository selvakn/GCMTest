package com.tw.pushlatency

import android.os.Bundle
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.LinearLayoutManager
import com.google.firebase.messaging.FirebaseMessaging
import com.tw.pushlatency.databinding.ActivityMainBinding
import com.tw.pushlatency.enrollment.EnrollmentManager
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.launch

class MainActivity : AppCompatActivity() {

    private lateinit var binding: ActivityMainBinding
    private lateinit var adapter: MessageAdapter

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)

        adapter = MessageAdapter(LatencyConfig.slowThresholdMillis(this))
        binding.messageList.layoutManager = LinearLayoutManager(this)
        binding.messageList.adapter = adapter

        observeMessages()
        enrollIfNeeded()
    }

    private fun observeMessages() {
        val app = application as PushLatencyApp
        lifecycleScope.launch {
            app.database.messageDao().observeAll().collectLatest { messages ->
                adapter.submitList(messages)
                // Empty state must be shown clearly rather than a blank screen (FR-025).
                binding.emptyState.visibility = if (messages.isEmpty()) android.view.View.VISIBLE else android.view.View.GONE
                binding.messageList.visibility = if (messages.isEmpty()) android.view.View.GONE else android.view.View.VISIBLE
            }
        }
    }

    private fun enrollIfNeeded() {
        FirebaseMessaging.getInstance().token.addOnCompleteListener { task ->
            if (!task.isSuccessful) {
                showEnrollmentFailure()
                return@addOnCompleteListener
            }
            val token = task.result
            when (EnrollmentManager(applicationContext).enrollIfNeeded(token)) {
                is EnrollmentManager.Result.AlreadyEnrolled ->
                    Toast.makeText(this, R.string.enrollment_already_active, Toast.LENGTH_SHORT).show()
                is EnrollmentManager.Result.Enrolled -> Unit
                is EnrollmentManager.Result.Failed -> showEnrollmentFailure()
            }
        }
    }

    private fun showEnrollmentFailure() {
        // Non-blocking: a transient toast, never a crash or a blocked UI (FR-020).
        Toast.makeText(this, R.string.enrollment_failed, Toast.LENGTH_SHORT).show()
    }
}
