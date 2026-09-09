# Лунобот-2: покрытие `fusefs/node_fuse`

- Claim: кастомная задача «покрыть тестами пакет `fusefs/node_fuse`», часть 1 из 1.
- Основание выбора: последний доступный отчёт Codecov для `main` `e40b44db248b72deb958a8fe6c70ac6bca31e349`; файл имел 8.89% покрытия (28/315 строк). Более свежий отчёт для быстро меняющегося `main` на момент выбора ещё не был опубликован Codecov.
- Изменение: добавлен Unix-only `node_fuse_test.go` с покрытием преобразований FUSE-атрибутов и errno, `Getattr` для staged-файла, `Readdir`, чтения и освобождения read-handle, writable `Open`/`Write`/`Fsync`/`Flush`/`Release`, `Statfs`, отказов записи и неподдерживаемого `Readlink`.
- Локально выполнены только `gofmt` и `git diff --check`; Go build/test не запускались согласно `LUNOBOT.md`.
- Hosted CI: ожидается для PR.
