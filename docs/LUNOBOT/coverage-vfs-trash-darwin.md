# Coverage: `vfs/trash_darwin.go`

Кастомная задача Lunobot-2: покрыть тестами Darwin-реализацию `OSVFS.MoveToTrash` и её Objective-C FFI-хелперы.

- На `main` Codecov: 62.45%; файл: 0.00% (83 строки).
- Тестовый коммит: `caaca38254c86dc65153b3da2508e4ba038e466b`.
- Коммит форматирования: `96a90a62dd5e2c2f90de7f0d7a3bec92dd7a4b39`.
- Тесты Darwin-only и не используют реальный системный Trash: Objective-C вызовы подменяются через тестовые stubs.
- Покрыты preflight/cancellation, успешные перемещения файла и каталога, ошибки Foundation/NSURL/NSFileManager/native operation и формирование native error description.
- CI #4002 был красным только из-за CRLF в тестовом файле; line endings нормализованы до LF.
- Статус: исправление форматирования отправлено, CI #4003 выполняется.