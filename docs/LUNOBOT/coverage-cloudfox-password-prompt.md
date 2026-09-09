# Coverage: `plugins/cloudfox/password_prompt.go`

Кастомная задача Lunobot-2: покрыть тестами UI-промпт master password для portable CloudFox vault.

- На `main` Codecov: 62.53%; файл: 0.00% (72 строки).
- Тестовый коммит: `28ed1a12fae999a53e8a50995350b76cee4834ed`.
- Покрыты отсутствие активного UI, отмена контекста, успешный round-trip через `FrameManager.PostTask`, принятие и очистка пароля, отмена, несовпадающее подтверждение и оба исхода предупреждения для пустого master password.
- Тесты используют только `NewSilentScreenBuf` и штатные callbacks vtui; реальные vault и терминал не затрагиваются.
- Статус: тесты отправлены, CI ожидается.