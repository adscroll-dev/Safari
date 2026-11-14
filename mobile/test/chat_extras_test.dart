// Tests for v0.7 chat extras: user-created folders, chat lock with a
// PIN, per-chat drafts, forwarding labels, voice playback speed.
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:vyro/chat_extras.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  group('forwardedLabel', () {
    test('shows the original sender name', () {
      expect(
        forwardedLabel({
          'forwarded_from': 'alice',
          'forwarded_from_id': 'u1',
        }),
        'Forwarded from alice',
      );
    });

    test('null when the message was not forwarded', () {
      expect(forwardedLabel({'content': 'hi'}), isNull);
      expect(forwardedLabel({'forwarded_from': '  '}), isNull);
    });

    test('trims the name', () {
      expect(
        forwardedLabel({'forwarded_from': '  bob  '}),
        'Forwarded from bob',
      );
    });
  });

  group('nextPlaybackSpeed', () {
    test('cycles 1x -> 1.5x -> 2x -> 1x', () {
      expect(nextPlaybackSpeed(1.0), 1.5);
      expect(nextPlaybackSpeed(1.5), 2.0);
      expect(nextPlaybackSpeed(2.0), 1.0);
    });

    test('formats speeds for the button', () {
      expect(formatPlaybackSpeed(1.0), '1x');
      expect(formatPlaybackSpeed(1.5), '1.5x');
      expect(formatPlaybackSpeed(2.0), '2x');
    });
  });

  group('ChatFolders', () {
    late SharedPreferences prefs;
    setUp(() async {
      SharedPreferences.setMockInitialValues({});
      prefs = await SharedPreferences.getInstance();
    });

    test('add / list / rename / remove with validation', () async {
      final folders = ChatFolders(prefs);
      expect(folders.list(), isEmpty);

      expect(await folders.add(''), 'Folder name is required');
      expect(await folders.add('   '), 'Folder name is required');
      expect(await folders.add('x' * 33),
          'Folder name too long (max 32 characters)');
      expect(await folders.add('Work'), isNull);
      expect(await folders.add('work'),
          'A folder with this name already exists');
      expect(await folders.add('Family'), isNull);
      expect(folders.list(), ['Work', 'Family']);

      expect(await folders.rename('Missing', 'Nope'), 'Folder not found');
      expect(await folders.rename('Work', 'Family'),
          'A folder with this name already exists');
      expect(await folders.rename('Work', 'Job'), isNull);
      expect(folders.list(), ['Job', 'Family']);

      await folders.remove('Job');
      expect(folders.list(), ['Family']);
    });

    test('corrupt stored data reads as empty, never crashes', () async {
      await prefs.setString('vyro_chat_folders_v1', 'not-json{{{');
      expect(ChatFolders(prefs).list(), isEmpty);
    });
  });

  group('ChatLock', () {
    late SharedPreferences prefs;
    setUp(() async {
      SharedPreferences.setMockInitialValues({});
      prefs = await SharedPreferences.getInstance();
    });

    test('PIN validation', () {
      expect(ChatLock.validatePin('123'), 'PIN must be 4-8 digits');
      expect(ChatLock.validatePin('123456789'), 'PIN must be 4-8 digits');
      expect(ChatLock.validatePin('12ab'), 'PIN must contain digits only');
      expect(ChatLock.validatePin('1234'), isNull);
      expect(ChatLock.validatePin('12345678'), isNull);
    });

    test('set / verify / lock / unlock / remove round-trip', () async {
      final lock = ChatLock(prefs);
      expect(lock.hasPin, isFalse);
      expect(await lock.verifyPin('1234'), isFalse);

      expect(await lock.setPin('12'), 'PIN must be 4-8 digits');
      expect(await lock.setPin('1234'), isNull);
      expect(lock.hasPin, isTrue);
      expect(await lock.verifyPin('1234'), isTrue);
      expect(await lock.verifyPin('0000'), isFalse);

      // The stored value is a hash, never the PIN itself.
      final stored = prefs.getString('vyro_chat_lock_pin_hash_v1')!;
      expect(stored.contains('1234'), isFalse);

      expect(lock.isLocked('chat1'), isFalse);
      await lock.lockChat('chat1');
      await lock.lockChat('chat2');
      expect(lock.isLocked('chat1'), isTrue);
      expect(lock.lockedChatIds(), {'chat1', 'chat2'});
      await lock.unlockChat('chat1');
      expect(lock.isLocked('chat1'), isFalse);
      expect(lock.isLocked('chat2'), isTrue);

      // Changing the PIN keeps the locked chats.
      expect(await lock.setPin('9876'), isNull);
      expect(await lock.verifyPin('9876'), isTrue);
      expect(await lock.verifyPin('1234'), isFalse);
      expect(lock.isLocked('chat2'), isTrue);

      await lock.removePin();
      expect(lock.hasPin, isFalse);
      expect(lock.isLocked('chat2'), isFalse);
    });

    test('same PIN hashes differently with a different salt', () async {
      final a = await ChatLock.hashPin('1234', 'salt-a');
      final b = await ChatLock.hashPin('1234', 'salt-b');
      expect(a, isNot(b));
      expect(await ChatLock.hashPin('1234', 'salt-a'), a);
    });
  });

  group('MessageDrafts', () {
    late SharedPreferences prefs;
    setUp(() async {
      SharedPreferences.setMockInitialValues({});
      prefs = await SharedPreferences.getInstance();
    });

    test('save / restore / clear per chat', () async {
      final drafts = MessageDrafts(prefs);
      expect(drafts.draftFor('c1'), isNull);
      await drafts.save('c1', 'hello there');
      await drafts.save('c2', 'other chat');
      expect(drafts.draftFor('c1'), 'hello there');
      expect(drafts.draftFor('c2'), 'other chat');
      // Saving empty text clears that chat's draft only.
      await drafts.save('c1', '   ');
      expect(drafts.draftFor('c1'), isNull);
      expect(drafts.draftFor('c2'), 'other chat');
      await drafts.clear('c2');
      expect(drafts.draftFor('c2'), isNull);
    });

    test('corrupt stored data reads as empty, never crashes', () async {
      await prefs.setString('vyro_message_drafts_v1', '[[broken');
      expect(MessageDrafts(prefs).draftFor('c1'), isNull);
    });
  });
}
