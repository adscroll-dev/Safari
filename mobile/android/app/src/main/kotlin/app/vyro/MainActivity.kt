package app.vyro

import android.Manifest
import android.content.pm.PackageManager
import android.os.Build
import android.content.Intent
import androidx.core.content.FileProvider
import androidx.core.content.ContextCompat
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import java.io.File

// Stage 8: a tiny MethodChannel for the system share sheet, so chat
// export needs no extra plugin (share_plus wants an Android Gradle
// plugin newer than this offline build can resolve). The Dart side
// calls "shareFile" with {path, text}; Android opens the chooser
// with the file attached as a content:// URI via FileProvider.
class MainActivity : FlutterActivity() {
    private val channelName = "safari/share"
    private val callChannelName = "safari/call_service"
    private var callChannel: MethodChannel? = null
    private var pendingCallChatId: String? = null

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(
            flutterEngine.dartExecutor.binaryMessenger,
            channelName
        ).setMethodCallHandler { call, result ->
            if (call.method == "shareFile") {
                val path = call.argument<String>("path")
                val text = call.argument<String>("text")
                if (path == null) {
                    result.error("NO_PATH", "shareFile needs a file path", null)
                    return@setMethodCallHandler
                }
                try {
                    val uri = FileProvider.getUriForFile(
                        this, "app.vyro.fileprovider", File(path)
                    )
                    val intent = Intent(Intent.ACTION_SEND).apply {
                        type = "text/plain"
                        putExtra(Intent.EXTRA_STREAM, uri)
                        if (text != null) putExtra(Intent.EXTRA_TEXT, text)
                        addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
                    }
                    startActivity(
                        Intent.createChooser(intent, "Share chat export")
                    )
                    result.success(null)
                } catch (e: Exception) {
                    result.error("SHARE_FAILED", e.message, null)
                }
            } else {
                result.notImplemented()
            }
        }
        pendingCallChatId = intent?.getStringExtra(CallForegroundService.EXTRA_CHAT_ID)
        callChannel = MethodChannel(
            flutterEngine.dartExecutor.binaryMessenger,
            callChannelName
        ).also { channel ->
            channel.setMethodCallHandler { call, result ->
                when (call.method) {
                    "startCall" -> {
                        val chatId = call.argument<String>("chatId")
                        if (chatId.isNullOrEmpty()) {
                            result.error("NO_CHAT_ID", "startCall needs a chat id", null)
                            return@setMethodCallHandler
                        }
                        try {
                            val intent = Intent(this, CallForegroundService::class.java).apply {
                                action = CallForegroundService.ACTION_START
                                putExtra(CallForegroundService.EXTRA_CHAT_ID, chatId)
                                putExtra(
                                    CallForegroundService.EXTRA_PEER_NAME,
                                    call.argument<String>("peerName").orEmpty()
                                )
                                putExtra(
                                    CallForegroundService.EXTRA_VIDEO,
                                    call.argument<Boolean>("video") == true
                                )
                            }
                            ContextCompat.startForegroundService(this, intent)
                        } catch (e: Exception) {
                            result.error("CALL_SERVICE_START_FAILED", e.message, null)
                            return@setMethodCallHandler
                        }
                        result.success(null)
                        requestNotificationPermissionIfNeeded()
                    }
                    "stopCall" -> {
                        stopService(Intent(this, CallForegroundService::class.java))
                        result.success(null)
                    }
                    "consumePendingCallNavigation" -> {
                        result.success(pendingCallChatId)
                        pendingCallChatId = null
                    }
                    else -> result.notImplemented()
                }
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        pendingCallChatId = intent.getStringExtra(CallForegroundService.EXTRA_CHAT_ID)
        pendingCallChatId?.let {
            callChannel?.invokeMethod("openCallScreen", it)
        }
    }

    private fun requestNotificationPermissionIfNeeded() {
        if (Build.VERSION.SDK_INT >= 33 &&
            ContextCompat.checkSelfPermission(
                this,
                Manifest.permission.POST_NOTIFICATIONS
            ) != PackageManager.PERMISSION_GRANTED
        ) {
            requestPermissions(
                arrayOf(Manifest.permission.POST_NOTIFICATIONS),
                REQUEST_POST_NOTIFICATIONS
            )
        }
    }

    companion object {
        private const val REQUEST_POST_NOTIFICATIONS = 1807
    }
}
