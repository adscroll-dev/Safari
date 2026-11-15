package app.vyro

import android.content.Intent
import androidx.core.content.FileProvider
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
    }
}
