import 'package:shared_preferences/shared_preferences.dart';

class E2eeChatState {
  const E2eeChatState(this._preferences);

  final SharedPreferences _preferences;

  String _key(String userId) => 'vyro_e2ee_chats_v1:$userId';

  Set<String> load(String userId) =>
      (_preferences.getStringList(_key(userId)) ?? <String>[]).toSet();

  Future<void> save(String userId, Set<String> chatIds) async {
    final saved =
        await _preferences.setStringList(_key(userId), chatIds.toList());
    if (!saved) {
      throw StateError('Could not save this account’s E2EE chat settings');
    }
  }
}
