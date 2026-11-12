# Dart and Flutter Testing

How the four layers map to a Dart/Flutter project, and the mechanics that differ from other stacks: widget trees, golden files, generated code and device-bound integration tests.

## Project shape

```text
pubspec.yaml
analysis_options.yaml          # include: package:flutter_lints/flutter.yaml
lib/                           # production code
test/                          # unit + widget tests (VM)
test/goldens/                  # golden images
integration_test/              # device/emulator tests
```

- Pure Dart package: `dart test` runs everything under `test/`.
- Flutter app/package: `flutter test` runs unit and widget tests on the Flutter test binary; `dart test` cannot pump widgets.
- `integration_test/` is executed with `flutter test integration_test -d <device>`, not by `flutter test` alone.

## Layers

| Layer | Location | Tool | Notes |
|---|---|---|---|
| **Unit** | `test/**/*_test.dart` | `dart test` / `flutter test` | Pure logic, no `WidgetTester` |
| **Integration (component)** | `test/` | `testWidgets` | A widget plus its real children, fake I/O |
| **Golden** | `test/` | `matchesGoldenFile` | Pixel comparison; platform-sensitive |
| **System** | `integration_test/` | `flutter test integration_test -d <device>` | Real app, device, slow |
| **Regression** | anywhere | same as the layer it grew from | Name it after the bug |

Tags work in both runners:

```dart
@Tags(["integration"])
library;

import "package:test/test.dart";
```

```bash
dart test --tags integration
flutter test --tags integration
```

Prefer directories when the project already has them; tags avoid moving files that mirror `lib/`.

## Widget tests

```dart
testWidgets("shows the empty state for a new account", (tester) async {
  await tester.pumpWidget(
    MaterialApp(home: InvoiceList(invoices: const [])),
  );
  expect(find.text("No invoices yet"), findsOneWidget);
  expect(find.byKey(const Key("invoice-list")), findsNothing);
});
```

Rules:

- **Find by role/key/text**, in that order; avoid `find.byType` for anything reusable.
- `pump()` advances one frame; `pumpAndSettle()` waits for animations — it **times out with infinite animations** (progress indicators, looping loaders). Use `pump(Duration)` with a bounded number of frames in that case.
- `tester.runAsync()` is required for real asynchronous I/O (file system, isolates); mock network calls instead.
- Reset surface size and text scale in tests that change them:
  `tester.view.physicalSize`, `tester.view.devicePixelRatio`, `MediaQuery.textScaler`.
- Localization needs the delegates: pump a `MaterialApp` with `localizationsDelegates` and `supportedLocales`, or text finders fail.
- Never `sleep`; use `fake_async` in unit tests and `pump` in widget tests.

## Fakes, mocks and dependency injection

| Need | Tool |
|---|---|
| Hand-written fake | A class implementing your repository interface (preferred) |
| Mocks without codegen | `mocktail` |
| Mocks with codegen | `mockito` + `build_runner` |
| Time control | `fake_async` (`FakeAsync().run(...)`) |
| HTTP | `http`'s `MockClient`, or a Dio adapter — never a real network call |

- Prefer constructor injection. Flutter makes it easy to pass a repository down the tree; do that before reaching for a service locator.
- If the app uses Riverpod, override at the root: `ProviderScope(overrides: [repoProvider.overrideWithValue(fake)])`.
- If it uses `get_it`, reset the locator in `tearDown` so tests stay independent.
- Do not mock widgets or `BuildContext`; test through the widget tree.

## Golden tests

```bash
flutter test --update-goldens test/invoice_card_test.dart
```

- Goldens compare pixels: fonts, anti-aliasing and platform rendering differ. Run them on the **same platform in CI** that generated them (usually Linux).
- Keep one canonical device size and text scale; changing them invalidates every golden.
- Treat an unexpected golden diff as a behavior change: review the image, do not blind-update.
- Golden tests are regression detectors for layout, not a substitute for behavior assertions.

## Integration tests

```dart
// integration_test/checkout_test.dart
IntegrationTestWidgetsFlutterBinding.ensureInitialized();

testWidgets("guest checkout completes", (tester) async {
  await tester.pumpWidget(const App());
  await tester.tap(find.text("Add to cart"));
  await tester.pumpAndSettle();
  expect(find.text("Order confirmed"), findsOneWidget);
});
```

- Run on a real device or emulator: `flutter test integration_test -d <device-id>`.
- Keep them few and on the critical journey; they are slow and device-bound.
- In CI, use an emulator action or a device farm; never run them on every pull request.

## Generated code

- `build_runner` output (`*.g.dart`, `*.freezed.dart`, `*.mocks.dart`) is not tested directly.
- Exclude it from analysis and coverage in `analysis_options.yaml`:

```yaml
analyzer:
  exclude:
    - "**/*.g.dart"
    - "**/*.freezed.dart"
```

- Test the generator's output through the feature that uses it, not by asserting on generated files.

## Coverage

```bash
# Flutter: writes coverage/lcov.info
flutter test --coverage

# Pure Dart: raw VM coverage, then convert to lcov
dart test --coverage=coverage
dart pub global run coverage:format_coverage \
  --lcov --in=coverage --out=coverage/lcov.info --report-on=lib
```

- Filter generated and bootstrap files out of the report before enforcing a threshold.
- `// coverage:ignore-line` and `// coverage:ignore-start` / `// coverage:ignore-end` cover the unavoidable gaps (platform channels, `main`).
- Same rule as everywhere: coverage is a diagnostic, not a target.

## CI commands

```bash
dart format --output=none --set-exit-if-changed .
dart analyze --fatal-infos
flutter test
flutter test --coverage
```

- `dart format` is the formatter; there is no separate `flutter format`.
- `dart analyze --fatal-infos` turns lints into failures; without it, warnings pass silently.
- Cache `~/.pub-cache` and use `flutter pub get --offline` when the lockfile is unchanged.
- Add `--no-pub` to `flutter test` after an explicit `pub get` in CI to avoid re-resolving on every step.

## Common pitfalls

| Pitfall | Symptom | Fix |
|---|---|---|
| `pumpAndSettle` with a looping animation | Test times out | Bounded `pump(Duration)` |
| Real network in a widget test | Slow, flaky, blocked in CI | `MockClient` / fake repository |
| Missing localization delegates | Text finders fail or throw | Pump a configured `MaterialApp` |
| Shared `get_it` state | Order-dependent failures | Reset in `setUp`/`tearDown` |
| Goldens generated on macOS, verified on Linux | Permanent diff | Pin the platform |
| Generated files analyzed/covered | Noise and impossible thresholds | `analyzer.exclude` + coverage filters |
| `dart test` on a Flutter widget test | "package:flutter not found" | `flutter test` |
| Integration tests in PR CI | 10+ minute pipelines, flaky devices | Nightly/release only |
