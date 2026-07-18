import 'package:flutter_test/flutter_test.dart';
import 'package:cricket_drs_mobile/main.dart';

void main() {
  testWidgets('app builds without crashing', (tester) async {
    await tester.pumpWidget(const CricketDrsApp());
    expect(find.text('Cricket DRS — Mobile (scaffold)'), findsOneWidget);
  });
}
