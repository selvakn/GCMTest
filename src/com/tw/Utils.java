package com.tw;

import android.content.Context;
import android.os.Handler;
import android.os.Looper;
import android.widget.Toast;

import java.util.concurrent.Callable;

import static java.lang.String.format;

public class Utils {
    public static void toast(final Context context, final String message, final Object... messageParts) {
        postToUiThread(new Callable() {
            @Override
            public Object call() throws Exception {
                Toast.makeText(context.getApplicationContext(), format(message, messageParts), Toast.LENGTH_LONG).show();
                return null;
            }
        });

    }

    private static void postToUiThread(final Callable c) {
        new Handler(Looper.getMainLooper()).post(new Runnable() {
            public void run() {
                try {
                    c.call();
                } catch (Exception ignored) {
                }
            }
        });
    }
}
