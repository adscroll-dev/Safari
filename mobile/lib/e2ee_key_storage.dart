import 'package:flutter/foundation.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';

abstract interface class SecureValueStorage {
  Future<String?> read(String key);
  Future<void> write(String key, String value);
}

class FlutterSecureValueStorage implements SecureValueStorage {
  const FlutterSecureValueStorage();

  static const FlutterSecureStorage _storage = FlutterSecureStorage();

  @override
  Future<String?> read(String key) => _storage.read(key: key);

  @override
  Future<void> write(String key, String value) =>
      _storage.write(key: key, value: value);
}

class E2eeKeyStorage {
  E2eeKeyStorage({
    required SharedPreferences preferences,
    required SecureValueStorage secureStorage,
    void Function(String message)? onWarning,
  })  : _preferences = preferences,
        _secureStorage = secureStorage,
        _onWarning = onWarning ?? debugPrint;

  final SharedPreferences _preferences;
  final SecureValueStorage _secureStorage;
  final void Function(String message) _onWarning;
  bool _warned = false;

  Future<String?> read(String key) async {
    String? secureValue;
    try {
      secureValue = await _secureStorage.read(key);
    } catch (error) {
      _warnUnavailable(error);
      return _preferences.getString(key);
    }
    if (secureValue != null) return secureValue;

    final legacyValue = _preferences.getString(key);
    if (legacyValue == null) return null;

    try {
      await _secureStorage.write(key, legacyValue);
      await _preferences.remove(key);
    } catch (error) {
      _warnUnavailable(error);
    }
    return legacyValue;
  }

  Future<void> write(String key, String value) async {
    try {
      await _secureStorage.write(key, value);
      if (_preferences.containsKey(key)) {
        await _preferences.remove(key);
      }
    } catch (error) {
      _warnUnavailable(error);
      await _preferences.setString(key, value);
    }
  }

  void _warnUnavailable(Object error) {
    if (_warned) return;
    _warned = true;
    _onWarning(
      'Secure storage unavailable; E2EE private keys are falling back to '
      'SharedPreferences. $error',
    );
  }
}
