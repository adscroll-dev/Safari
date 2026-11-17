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

class E2eeSecureStorageException implements Exception {
  const E2eeSecureStorageException(this.message);

  final String message;

  @override
  String toString() => message;
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
      throw const E2eeSecureStorageException(
        'E2EE is unavailable because secure key storage could not be accessed.',
      );
    }
    if (secureValue != null) return secureValue;

    final legacyValue = _preferences.getString(key);
    if (legacyValue == null) return null;

    try {
      await _secureStorage.write(key, legacyValue);
      final migratedValue = await _secureStorage.read(key);
      if (migratedValue != legacyValue) {
        throw StateError('Secure storage did not verify the migrated key');
      }
      await _preferences.remove(key);
      return migratedValue;
    } catch (error) {
      _warnUnavailable(error);
      throw const E2eeSecureStorageException(
        'E2EE is unavailable because the legacy key could not be migrated '
        'and verified in secure storage.',
      );
    }
  }

  Future<void> write(String key, String value) async {
    try {
      await _secureStorage.write(key, value);
      if (await _secureStorage.read(key) != value) {
        throw StateError('Secure storage did not verify the written key');
      }
      if (_preferences.containsKey(key)) {
        await _preferences.remove(key);
      }
    } catch (error) {
      _warnUnavailable(error);
      throw const E2eeSecureStorageException(
        'E2EE is unavailable because the private key could not be stored '
        'and verified securely.',
      );
    }
  }

  void _warnUnavailable(Object error) {
    if (_warned) return;
    _warned = true;
    _onWarning(
      'Secure storage unavailable; E2EE is disabled until private keys can '
      'be stored securely. $error',
    );
  }
}
