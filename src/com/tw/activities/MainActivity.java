package com.tw.activities;

import android.app.Activity;
import android.os.Bundle;
import android.util.Log;
import com.google.android.gcm.GCMRegistrar;
import com.tw.R;

public class MainActivity extends Activity
{
    private static final String TAG = "GCM_TEST";
    private static final String SENDER_ID = "692730889083";

    @Override
    public void onCreate(Bundle savedInstanceState)
    {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.main);

        GCMRegistrar.checkDevice(this);
        GCMRegistrar.checkManifest(this);
        final String regId = GCMRegistrar.getRegistrationId(this);
        if (regId.equals("")) {
          GCMRegistrar.register(this, SENDER_ID);
        } else {
          Log.v(TAG, "Already registered");
        }
    }
}
