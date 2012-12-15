package com.tw.activities;

import android.app.Activity;
import android.os.Bundle;
import com.google.android.gcm.GCMRegistrar;
import com.tw.GCMIntentService;
import com.tw.R;

import static com.tw.Utils.toast;

public class MainActivity extends Activity {
    @Override
    public void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.main);

        GCMRegistrar.checkDevice(this);
        GCMRegistrar.checkManifest(this);
        final String regId = GCMRegistrar.getRegistrationId(this);
        if (regId.equals("")) {
            GCMRegistrar.register(this, GCMIntentService.SENDER_ID);
        } else {
            toast(this, "Already registered with GCM");
        }
    }
}
