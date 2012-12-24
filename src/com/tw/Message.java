package com.tw;

import android.graphics.Color;
import android.text.Spannable;
import android.text.format.DateFormat;
import android.text.style.ForegroundColorSpan;
import android.util.Log;

import java.util.Date;

public class Message {
    public static final String ID = "id";
    public static final String SENT_TIMESTAMP = "sentTimestamp";
    public static final String RECEIVE_TIMESTAMP = "receiveTimestamp";
    public static final String LATENCY_IN_MILLIS = "latency";
    public static final int TWO_SECONDS = 2000;
    public final CharSequence id;
    public final CharSequence sentTimestamp;
    public final CharSequence receivedTimestamp;
    public final CharSequence latency;

    public Message(CharSequence id, Date sentTimestamp, Date receivedTimestamp) {
        this.id = computeId(id);

        this.sentTimestamp = DateFormat.format("kk:mm:ss", sentTimestamp);

        this.receivedTimestamp = DateFormat.format("kk:mm:ss", receivedTimestamp);
        this.latency = computeLatency(receivedTimestamp, sentTimestamp);
        Log.i("Message", "Sent: " + this.sentTimestamp + ", Received: " + this.receivedTimestamp);
    }

    private CharSequence computeLatency(Date receivedTimestamp, Date sentTimestamp) {
        long diff = receivedTimestamp.getTime() - sentTimestamp.getTime();
        Spannable diffSpan = Spannable.Factory.getInstance().newSpannable(diff <= 0 ? "Too Low" : String.valueOf(diff));
        diffSpan.setSpan(new ForegroundColorSpan(diff > TWO_SECONDS ? Color.RED: Color.GREEN), 0, diffSpan.length(), 0);
        return diffSpan;
    }

    private static CharSequence computeId(CharSequence id) {
        Spannable idSpan = Spannable.Factory.getInstance().newSpannable(id);
        idSpan.setSpan(new ForegroundColorSpan(Color.RED), 0, id.length(), 0);
        return idSpan;

    }

}
