# Coverage: `vfs/trash_darwin.go`

Кастомная задача Lunobot-2: покрыть тестами Darwin-реализацию `OSVFS.MoveToTrash` и её Objective-C FFI-хелперы.

- На `main` Codecov: 62.45%; файл: 0.00% (83 строки).
- Коммит с тестами: `caaca38254c86dc65153b3da2508e4ba038e466b`.
- Тесты Darwin-only и не используют реальный системный Trash: Objective-C вызовы подменяются через тестовые stubs.
- Покрыты preflight/cancellation, успешные перемещение файла и каталога, ошибки Foundation/NSURL/NSFileManager/native operation и формирование native error description.
- Статус: тестовый коммит создан; CI и PR ещё не запущены.