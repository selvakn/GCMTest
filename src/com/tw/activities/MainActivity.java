package com.tw.activities;

import android.app.ListActivity;
import android.content.Intent;
import android.os.Bundle;
import android.util.Log;
import android.view.View;
import android.view.ViewGroup;
import android.widget.ArrayAdapter;
import android.widget.TextView;
import com.google.android.gcm.GCMRegistrar;
import com.tw.GCMIntentService;
import com.tw.Message;
import com.tw.R;

import java.util.Date;

import static com.tw.Message.RECEIVE_TIMESTAMP;
import static com.tw.Message.SENT_TIMESTAMP;
import static com.tw.Message.ID;
import static com.tw.Utils.toast;

public class MainActivity extends ListActivity {

    private ArrayAdapter<Message> adapter;

    @Override
    public void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.main);
        getListView().setEmptyView(findViewById(R.id.empty_view));
        adapter = new ArrayAdapter<Message>(this, R.layout.message_item) {
            @Override
            public View getView(int position, View convertView, ViewGroup parent) {
                View view = convertView == null ? getLayoutInflater().inflate(R.layout.message_item, null) : convertView;
                Message model = getItem(position);
                TextView idView = (TextView) view.findViewById(R.id.message_id);
                TextView receivedView = (TextView) view.findViewById(R.id.received_timestamp);
                TextView sentView = (TextView) view.findViewById(R.id.sent_timestamp);
                TextView latencyView = (TextView) view.findViewById(R.id.latency);

                idView.setText(model.id);
                receivedView.setText(model.receivedTimestamp);
                sentView.setText(model.sentTimestamp);
                latencyView.setText(model.latency);
                return view;
            }
        };
        setListAdapter(adapter);

        registerWithGCMIfNecessary();
    }

    private void registerWithGCMIfNecessary() {
        GCMRegistrar.checkDevice(this);
        GCMRegistrar.checkManifest(this);
        final String regId = GCMRegistrar.getRegistrationId(this);
        if (regId.equals("")) {
            GCMRegistrar.register(this, GCMIntentService.SENDER_ID);
        } else {
            toast(this, "Already registered with GCM");
        }
    }

    @Override
    protected void onNewIntent(Intent intent) {
        super.onNewIntent(intent);
        String id = intent.getStringExtra(ID);
        long sent = intent.getLongExtra(SENT_TIMESTAMP, 0);
        long received = intent.getLongExtra(RECEIVE_TIMESTAMP, 0);
        Log.i("Main", "Received id: " + id + ", timestamp " + sent);
        adapter.insert(new Message(id, new Date(received), new Date(sent)), 0);
    }
}
