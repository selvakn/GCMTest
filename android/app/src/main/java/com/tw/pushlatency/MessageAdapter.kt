package com.tw.pushlatency

import android.graphics.Color
import android.view.LayoutInflater
import android.view.ViewGroup
import androidx.recyclerview.widget.RecyclerView
import com.tw.pushlatency.data.MessageEntity
import com.tw.pushlatency.databinding.ItemMessageBinding
import java.text.DateFormat
import java.util.Date

/** Most-recent-first list of received rounds (FR-024), with latency visually
 * categorized as good/slow/anomalous rather than shown as a raw number (FR-026). */
class MessageAdapter(private val slowThresholdMillis: Long) :
    RecyclerView.Adapter<MessageAdapter.ViewHolder>() {

    private var items: List<MessageEntity> = emptyList()

    fun submitList(newItems: List<MessageEntity>) {
        items = newItems
        notifyDataSetChanged()
    }

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ViewHolder {
        val binding = ItemMessageBinding.inflate(LayoutInflater.from(parent.context), parent, false)
        return ViewHolder(binding)
    }

    override fun onBindViewHolder(holder: ViewHolder, position: Int) {
        holder.bind(items[position], slowThresholdMillis)
    }

    override fun getItemCount(): Int = items.size

    class ViewHolder(private val binding: ItemMessageBinding) : RecyclerView.ViewHolder(binding.root) {
        private val dateFormat = DateFormat.getTimeInstance(DateFormat.MEDIUM)

        fun bind(message: MessageEntity, slowThresholdMillis: Long) {
            binding.roundId.text = binding.root.context.getString(R.string.round_label, message.roundId)
            binding.times.text = binding.root.context.getString(
                R.string.sent_received_label,
                dateFormat.format(Date(message.sentAtMillis)),
                dateFormat.format(Date(message.receivedAtMillis)),
            )

            val category = LatencyConfig.categorize(message.latencyMillis, slowThresholdMillis)
            val (label, color) = when (category) {
                LatencyCategory.GOOD -> "${message.latencyMillis} ms — good" to Color.parseColor("#2E7D32")
                LatencyCategory.SLOW -> "${message.latencyMillis} ms — slow" to Color.parseColor("#F9A825")
                LatencyCategory.ANOMALOUS -> "${message.latencyMillis} ms — anomalous" to Color.parseColor("#C62828")
            }
            binding.latency.text = label
            binding.latency.setTextColor(color)
        }
    }
}
