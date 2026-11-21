import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vyro/main.dart';
import 'package:vyro/theme.dart';

void main() {
  testWidgets('Chats filters and section headers render Persian labels',
      (WidgetTester tester) async {
    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('fa'),
        supportedLocales: const [Locale('en'), Locale('fa'), Locale('ar')],
        localizationsDelegates: const [
          GlobalMaterialLocalizations.delegate,
          GlobalWidgetsLocalizations.delegate,
          GlobalCupertinoLocalizations.delegate,
        ],
        home: Builder(
          builder: (context) => Scaffold(
            body: Column(
              children: [
                SafariFilterChip(
                  label: chatPageText(context, 'Channels'),
                  selected: true,
                  onTap: () {},
                ),
                Text(chatPageText(context, 'Pinned')),
                Text(chatPageText(context, 'Chats')),
              ],
            ),
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.text('کانال‌ها'), findsOneWidget);
    expect(find.text('سنجاق‌شده'), findsOneWidget);
    expect(find.text('گفت‌وگوها'), findsOneWidget);
    expect(find.text('Channels'), findsNothing);
    expect(find.text('Pinned'), findsNothing);
    expect(find.text('Chats'), findsNothing);
  });
}
