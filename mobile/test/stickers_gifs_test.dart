import 'dart:io';
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vyro/api.dart';
import 'package:vyro/stickers_gifs_picker.dart';

class _CatalogApi extends ApiClient {
  _CatalogApi({required this.stickerBytes, required this.gifBytes})
      : super(baseUrl: 'http://localhost');

  final Uint8List stickerBytes;
  final Uint8List gifBytes;
  final List<Map<String, dynamic>> sent = [];
  final List<String> downloaded = [];

  @override
  Future<List<dynamic>> listStickerPacks() async => [
        {
          'id': 'safari-expressions',
          'title': 'Safari Expressions',
          'sticker_count': 11,
        },
      ];

  @override
  Future<Map<String, dynamic>> getStickerPack(String id) async => {
        'id': id,
        'title': 'Safari Expressions',
        'stickers': [
          {
            'id': 'happy',
            'name': 'Happy',
            'file_url': '/stickers/happy.webp',
            'size': 100,
          },
        ],
      };

  @override
  Future<List<dynamic>> listGifs([String query = '']) async => [
        {
          'id': 'planet-spin',
          'name': 'Spinning Safari Planet',
          'file_url': '/gifs/planet-spin.gif',
          'size': 100,
        },
      ]
          .where((item) => (item['name'] as String)
              .toLowerCase()
              .contains(query.trim().toLowerCase()))
          .toList();

  @override
  Future<Uint8List> downloadFile(String fileUrl) async {
    downloaded.add(fileUrl);
    return fileUrl.startsWith('/gifs/') ? gifBytes : stickerBytes;
  }

  @override
  Future<Map<String, dynamic>> sendCatalogMediaMessage(
    String chatId,
    Map<String, dynamic> item, {
    required String kind,
  }) async {
    final message = {
      'chat_id': chatId,
      'kind': kind,
      'file_url': item['file_url'],
    };
    sent.add(message);
    return message;
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  late Uint8List stickerBytes;
  late Uint8List gifBytes;

  setUpAll(() async {
    stickerBytes = await File(
      '../backend/internal/handlers/static/stickers/happy.webp',
    ).readAsBytes();
    gifBytes = await File(
      '../backend/internal/handlers/static/gifs/planet-spin.gif',
    ).readAsBytes();
  });

  Future<void> openPicker(WidgetTester tester, _CatalogApi api) async {
    await tester.pumpWidget(MaterialApp(
      home: Builder(
        builder: (context) => Scaffold(
          body: ElevatedButton(
            onPressed: () => showModalBottomSheet<void>(
              context: context,
              builder: (context) => SizedBox(
                height: 500,
                child: StickersGifsPicker(
                  api: api,
                  chatId: 'chat-1',
                  dataSaver: true,
                  emojis: const ['🙂'],
                  onEmoji: (_) {},
                  onSent: (_) {},
                  onDownloaded: (_, __) {},
                ),
              ),
            ),
            child: const Text('Open picker'),
          ),
        ),
      ),
    ));
    await tester.tap(find.text('Open picker'));
    await tester.pumpAndSettle();
  }

  testWidgets('sticker tab shows the real pack and sends its catalog URL',
      (tester) async {
    final api = _CatalogApi(stickerBytes: stickerBytes, gifBytes: gifBytes);
    await openPicker(tester, api);
    await tester.tap(find.text('Stickers'));
    await tester.pumpAndSettle();

    expect(find.text('Safari Expressions'), findsOneWidget);
    expect(find.byKey(const ValueKey('catalog-item-happy')), findsOneWidget);
    expect(api.downloaded, isEmpty);

    await tester.tap(find.byKey(const ValueKey('catalog-item-happy')));
    await tester.pumpAndSettle();
    expect(api.downloaded, ['/stickers/happy.webp']);
    await tester.tap(find.byKey(const ValueKey('catalog-item-happy')));
    await tester.pumpAndSettle();
    expect(api.sent, [
      {
        'chat_id': 'chat-1',
        'kind': 'sticker',
        'file_url': '/stickers/happy.webp',
      }
    ]);
  });

  testWidgets('GIF tab sends the selected real catalog URL', (tester) async {
    final api = _CatalogApi(stickerBytes: stickerBytes, gifBytes: gifBytes);
    await openPicker(tester, api);
    await tester.tap(find.text('GIFs'));
    await tester.pumpAndSettle();

    expect(
        find.byKey(const ValueKey('catalog-item-planet-spin')), findsOneWidget);
    expect(api.downloaded, isEmpty);
    await tester.tap(find.byKey(const ValueKey('catalog-item-planet-spin')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 150));
    expect(api.downloaded, ['/gifs/planet-spin.gif']);
    await tester.tap(find.byKey(const ValueKey('catalog-item-planet-spin')));
    await tester.pumpAndSettle();
    expect(api.sent.single['kind'], 'gif');
    expect(api.sent.single['file_url'], '/gifs/planet-spin.gif');
  });

  testWidgets('catalog GIF decodes as an animated Image.memory source',
      (tester) async {
    await tester.pumpWidget(MaterialApp(home: Image.memory(gifBytes)));
    final image = tester.widget<Image>(find.byType(Image));
    expect(image.image, isA<MemoryImage>());
    final codec = await ui.instantiateImageCodec(gifBytes);
    expect(codec.frameCount, greaterThan(1));
    codec.dispose();
  });
}
