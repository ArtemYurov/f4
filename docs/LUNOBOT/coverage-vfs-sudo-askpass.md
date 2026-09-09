# Лунобот-2: покрытие `vfs/sudo_askpass_unix`

- Claim: кастомная задача «покрыть тестами пакет vfs/sudo_askpass_unix», часть 1 из 1.
- Основание выбора: отчёт Codecov для `main` `e40b44db248b72deb958a8fe6c70ac6bca31e349`; файл имел 3.64% покрытия (4/110 строк). Более свежий отчёт для быстро меняющегося `main` ещё не опубликован Codecov.
- Изменение: добавлены Unix-only тесты subprocess askpass helper, Unix socket handshake, missing-parent exit, listen failure, attempt limit и nil FrameManager path.
- Локально выполнены только `gofmt` и `git diff --check`; Go build/test не запускались согласно `LUNOBOT.md`.
- Hosted CI: ожидается для PR.
