# Лунобот-2: покрытие `vfs/sudo_askpass_unix`

- Claim: кастомная задача «покрыть тестами пакет vfs/sudo_askpass_unix», часть 1 из 1.
- Основание выбора: отчёт Codecov для `main` `e40b44db248b72deb958a8fe6c70ac6bca31e349`; файл имел 3.64% покрытия (4/110 строк). Более свежий отчёт для быстро меняющегося `main` ещё не опубликован Codecov.
- Изменение: добавлены Unix-only тесты subprocess askpass helper, Unix socket handshake, missing-parent exit, listen failure, attempt limit и nil FrameManager path.
- Локально выполнены только `gofmt` и `git diff --check`; Go build/test не запускались согласно `LUNOBOT.md`.
- Hosted CI run `34334340207` на exact head `a3065179dc0e81ba5d9682712f40412b532ee154` завершился failure: `Lint (rest)` из-за G204 на запуске текущего test binary, а `Test (linux/amd64)`, `Test (linux/arm64)` и `Race (packages)` зависли в `TestAskpassServerReturnsWhenSocketCannotBeCreated` и завершились по timeout.
- Исправление: путь для listen-failure теперь занят каталогом, а намеренный запуск текущего test binary помечен `#nosec G204`.
- Следующий hosted CI: ожидается после исправляющего коммита.
