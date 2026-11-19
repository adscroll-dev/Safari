package app.vyro

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Intent
import android.content.pm.ServiceInfo
import android.os.Build
import android.os.IBinder
import androidx.core.app.NotificationCompat

class CallForegroundService : Service() {
    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (intent?.action == ACTION_STOP) {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.N) {
                stopForeground(STOP_FOREGROUND_REMOVE)
            } else {
                @Suppress("DEPRECATION")
                stopForeground(true)
            }
            stopSelf(startId)
            return START_NOT_STICKY
        }

        createNotificationChannel()
        val video = intent?.getBooleanExtra(EXTRA_VIDEO, false) == true
        val peerName = intent?.getStringExtra(EXTRA_PEER_NAME).orEmpty()
        val chatId = intent?.getStringExtra(EXTRA_CHAT_ID).orEmpty()
        val notification = buildNotification(peerName, video, chatId)
        // The foreground-service type must match THIS call, not the
        // Manifest's declared maximum: a voice call uses MICROPHONE
        // only, a video call uses MICROPHONE | CAMERA. The
        // three-parameter startForeground() exists since API 29 (Q), so
        // use it there and above; below API 29 the two-parameter call
        // falls back to the Manifest types (both), which is the only
        // option on those versions.
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            var serviceType = ServiceInfo.FOREGROUND_SERVICE_TYPE_MICROPHONE
            if (video) {
                serviceType = serviceType or ServiceInfo.FOREGROUND_SERVICE_TYPE_CAMERA
            }
            startForeground(NOTIFICATION_ID, notification, serviceType)
        } else {
            startForeground(NOTIFICATION_ID, notification)
        }
        return START_NOT_STICKY
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                "Active calls",
                NotificationManager.IMPORTANCE_LOW
            ).apply {
                description = "Ongoing voice and video calls"
                setShowBadge(false)
            }
            getSystemService(NotificationManager::class.java)
                .createNotificationChannel(channel)
        }
    }

    private fun buildNotification(
        peerName: String,
        video: Boolean,
        chatId: String
    ): Notification {
        val openIntent = Intent(this, MainActivity::class.java).apply {
            action = ACTION_OPEN_CALL
            putExtra(EXTRA_CHAT_ID, chatId)
            flags = Intent.FLAG_ACTIVITY_SINGLE_TOP or Intent.FLAG_ACTIVITY_CLEAR_TOP
        }
        val pendingIntent = PendingIntent.getActivity(
            this,
            0,
            openIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )
        val type = if (video) "Video call" else "Voice call"
        val content = if (peerName.isBlank()) type else "$type with $peerName"
        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setSmallIcon(R.mipmap.ic_launcher)
            .setContentTitle("Call in progress")
            .setContentText(content)
            .setCategory(NotificationCompat.CATEGORY_CALL)
            .setOngoing(true)
            .setOnlyAlertOnce(true)
            .setContentIntent(pendingIntent)
            .setPriority(NotificationCompat.PRIORITY_LOW)
            .build()
    }

    companion object {
        const val ACTION_START = "app.vyro.call.START"
        const val ACTION_STOP = "app.vyro.call.STOP"
        const val EXTRA_CHAT_ID = "app.vyro.call.CHAT_ID"
        const val EXTRA_PEER_NAME = "app.vyro.call.PEER_NAME"
        const val EXTRA_VIDEO = "app.vyro.call.VIDEO"
        private const val ACTION_OPEN_CALL = "app.vyro.call.OPEN"
        private const val CHANNEL_ID = "active_calls"
        private const val NOTIFICATION_ID = 1807
    }
}
