import 'dart:typed_data';

import 'package:flutter/material.dart';

import 'api.dart';

class StickersGifsPicker extends StatefulWidget {
  const StickersGifsPicker({
    super.key,
    required this.api,
    required this.chatId,
    required this.dataSaver,
    required this.emojis,
    required this.onEmoji,
    required this.onSent,
    required this.onDownloaded,
  });

  final ApiClient api;
  final String chatId;
  final bool dataSaver;
  final List<String> emojis;
  final ValueChanged<String> onEmoji;
  final ValueChanged<Map<String, dynamic>> onSent;
  final void Function(String fileUrl, Uint8List bytes) onDownloaded;

  @override
  State<StickersGifsPicker> createState() => _StickersGifsPickerState();
}

class _StickersGifsPickerState extends State<StickersGifsPicker>
    with SingleTickerProviderStateMixin {
  late final TabController _tabs;
  late Future<Map<String, dynamic>> _stickers;
  late Future<List<dynamic>> _gifs;
  final Map<String, Uint8List> _bytes = {};
  final Set<String> _downloading = {};
  final TextEditingController _search = TextEditingController();
  String? _error;

  @override
  void initState() {
    super.initState();
    _tabs = TabController(length: 3, vsync: this);
    _stickers = _loadStickers();
    _gifs = widget.api.listGifs();
  }

  @override
  void dispose() {
    _tabs.dispose();
    _search.dispose();
    super.dispose();
  }

  Future<Map<String, dynamic>> _loadStickers() async {
    final packs = await widget.api.listStickerPacks();
    if (packs.isEmpty) return {};
    final id = (packs.first as Map<String, dynamic>)['id'] as String;
    return widget.api.getStickerPack(id);
  }

  Future<void> _download(String fileUrl) async {
    if (_bytes.containsKey(fileUrl) || !_downloading.add(fileUrl)) return;
    setState(() => _error = null);
    try {
      final bytes = await widget.api.downloadFile(fileUrl);
      if (!mounted) return;
      setState(() {
        _bytes[fileUrl] = bytes;
        _downloading.remove(fileUrl);
      });
      widget.onDownloaded(fileUrl, bytes);
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _downloading.remove(fileUrl);
        _error = error.toString();
      });
    }
  }

  Future<void> _send(Map<String, dynamic> item, String kind) async {
    try {
      final message = await widget.api.sendCatalogMediaMessage(
        widget.chatId,
        item,
        kind: kind,
      );
      if (!mounted) return;
      widget.onSent(message);
      Navigator.of(context).pop();
    } catch (error) {
      if (mounted) setState(() => _error = error.toString());
    }
  }

  void _searchGifs(String query) {
    setState(() {
      _error = null;
      _gifs = widget.api.listGifs(query);
    });
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        TabBar(
          controller: _tabs,
          tabs: const [
            Tab(text: 'Emoji'),
            Tab(text: 'Stickers'),
            Tab(text: 'GIFs'),
          ],
        ),
        if (_error != null)
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
            child: Text(_error!, style: const TextStyle(color: Colors.red)),
          ),
        Expanded(
          child: TabBarView(
            controller: _tabs,
            children: [
              _emojiGrid(),
              _stickerGrid(),
              _gifGrid(),
            ],
          ),
        ),
      ],
    );
  }

  Widget _emojiGrid() => GridView.count(
        crossAxisCount: 6,
        padding: const EdgeInsets.all(12),
        children: [
          for (final emoji in widget.emojis)
            IconButton(
              key: ValueKey('composer-emoji-$emoji'),
              onPressed: () {
                widget.onEmoji(emoji);
                Navigator.of(context).pop();
              },
              icon: Text(emoji, style: const TextStyle(fontSize: 27)),
            ),
        ],
      );

  Widget _stickerGrid() => FutureBuilder<Map<String, dynamic>>(
        future: _stickers,
        builder: (context, snapshot) {
          if (snapshot.hasError) {
            return Center(child: Text(snapshot.error.toString()));
          }
          if (!snapshot.hasData) {
            return const Center(child: CircularProgressIndicator());
          }
          final pack = snapshot.data!;
          final items = (pack['stickers'] as List<dynamic>?) ?? const [];
          if (items.isEmpty) return const Center(child: Text('No stickers'));
          return Column(
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 10, 16, 4),
                child: Align(
                  alignment: Alignment.centerLeft,
                  child: Text(
                    pack['title'] as String? ?? '',
                    style: const TextStyle(fontWeight: FontWeight.w600),
                  ),
                ),
              ),
              Expanded(
                child: _catalogGrid(items, 'sticker'),
              ),
            ],
          );
        },
      );

  Widget _gifGrid() => Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
            child: TextField(
              controller: _search,
              decoration: const InputDecoration(
                prefixIcon: Icon(Icons.search),
                hintText: 'Search GIFs',
                isDense: true,
                border: OutlineInputBorder(),
              ),
              onChanged: _searchGifs,
            ),
          ),
          Expanded(
            child: FutureBuilder<List<dynamic>>(
              future: _gifs,
              builder: (context, snapshot) {
                if (snapshot.hasError) {
                  return Center(child: Text(snapshot.error.toString()));
                }
                if (!snapshot.hasData) {
                  return const Center(child: CircularProgressIndicator());
                }
                if (snapshot.data!.isEmpty) {
                  return const Center(child: Text('No GIFs found'));
                }
                return _catalogGrid(snapshot.data!, 'gif');
              },
            ),
          ),
        ],
      );

  Widget _catalogGrid(List<dynamic> items, String kind) => GridView.builder(
        padding: const EdgeInsets.all(8),
        gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
          crossAxisCount: 4,
          crossAxisSpacing: 6,
          mainAxisSpacing: 6,
        ),
        itemCount: items.length,
        itemBuilder: (context, index) {
          final item = items[index] as Map<String, dynamic>;
          final fileUrl = item['file_url'] as String;
          final bytes = _bytes[fileUrl];
          if (bytes == null &&
              !widget.dataSaver &&
              !_downloading.contains(fileUrl)) {
            WidgetsBinding.instance.addPostFrameCallback((_) {
              if (mounted) _download(fileUrl);
            });
          }
          return InkWell(
            key: ValueKey('catalog-item-${item['id']}'),
            onTap: () {
              if (bytes == null && widget.dataSaver) {
                _download(fileUrl);
              } else {
                _send(item, kind);
              }
            },
            child: DecoratedBox(
              decoration: BoxDecoration(
                color: Theme.of(context).colorScheme.surfaceContainerHighest,
                borderRadius: BorderRadius.circular(8),
              ),
              child: bytes == null
                  ? Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        if (_downloading.contains(fileUrl))
                          const SizedBox(
                            width: 20,
                            height: 20,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        else
                          const Icon(Icons.download_outlined),
                        const SizedBox(height: 4),
                        Text(
                          _downloading.contains(fileUrl)
                              ? 'Loading'
                              : 'Download',
                          style: const TextStyle(fontSize: 10),
                        ),
                      ],
                    )
                  : Image.memory(bytes, fit: BoxFit.contain),
            ),
          );
        },
      );
}
