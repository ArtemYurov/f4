# Покрытие `internal/app/bootstrap_settings.go`

Задача взята по § 22 п. 3: на main Codecov показывал 10.00% покрытия файла (80 строк), общее покрытие — 62.14%.

Добавлен `internal/app/bootstrap_settings_test.go`: тестируются локализованный label автоматического backend’а, сохранение явного имени backend’а и построение startup settings dialog с проверкой layout на silent frame manager.

CI выявил конфликт имени с существующим TestStartupBackendLabels; новый тест переименован в TestBootstrapStartupBackendLabels (коммит 1934959).

CI также выявил форматное расхождение в import-блоке; стандартный и внешний импорты объединены в коммите 1333599.
