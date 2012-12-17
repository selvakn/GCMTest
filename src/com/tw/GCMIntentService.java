package com.tw;

import android.content.Context;
import android.content.Intent;
import android.os.Bundle;
import com.google.android.gcm.GCMBaseIntentService;
import com.google.android.gcm.GCMRegistrar;
import org.apache.http.client.HttpClient;
import org.apache.http.client.methods.HttpGet;
import org.apache.http.impl.client.DefaultHttpClient;

import java.io.IOException;

import static com.tw.Utils.toast;
import static java.lang.String.format;

public class GCMIntentService extends GCMBaseIntentService
{

    public static final String ERROR_SENDING_STATISTICS = "Error sending statistics";
    public static final String SENDER_ID = "692730889083";
    public static final String STATISTICS_ENDPOINT = "http://192.168.2.101:4567";

    public GCMIntentService() {
        super(SENDER_ID);
    }

    @Override
    protected void onMessage(Context context, Intent intent) {
        toast(this, "Message Received");
        Bundle bundleExtras = intent.getExtras();
        String id = bundleExtras.getString("id");
        performGet(format("/received/%s/%s", GCMRegistrar.getRegistrationId(this), id));
    }

    @Override
    protected void onError(Context context, String message) {
        toast(this, "Error: %s", message);
    }

    @Override
    protected void onRegistered(Context context, String registrationId) {
        performGet(format("/register/%s", registrationId));
        toast(this, "Device Registered");
    }

    @Override
    protected void onUnregistered(Context context, String s) {

    }

    private void performGet(String url) {
        HttpClient client = new DefaultHttpClient();
        HttpGet httpGet = new HttpGet(STATISTICS_ENDPOINT + url);
        try {
            client.execute(httpGet);
        } catch (IOException e) {
            toast(this, ERROR_SENDING_STATISTICS);
        }
    }
}
