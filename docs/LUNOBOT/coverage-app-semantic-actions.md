# Coverage: internal/app/semantic.go

Задача: кастомное повышение покрытия по § 22 п. 3 LUNOBOT.md. На Codecov файл имел 0.00% покрытия при 90 строках.

Добавлены детерминированные тесты для semantic actions виджетов (edit, checkbox, radio group, list box и combo box), поиска target среди дочерних элементов, обработки frame actions и nil-action.

Локальные Go-сборки и тесты не запускались по инструкции Лунобота; результат проверяется через GitHub Actions.
