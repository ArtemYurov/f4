# Coverage: `internal/terminal/session_unix.go`

Кастомная задача Lunobot-2 по §22 п.3: расширить покрытие Unix session lifecycle и IPC helper-логики.

- На `main` Codecov: 55.82%; файл: 0.00% (444 строки).
- Тестовый коммит: `d191a443027dd30cb01402f8dfa90a33af4e34e2`.
- Покрыты live/stale/malformed session metadata, запись и очистка session info, invalid-descriptor error paths, truncation startup log, missing-server client failure и дополнительные варианты attach payload.
- Тесты не запускают настоящий daemon и не требуют интерактивного терминала.
- Статус: тесты отправлены, CI ожидается.