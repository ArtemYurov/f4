# Coverage: internal/gui/icon_darwin.go

## Статус

- [x] Покрыты ранние выходы applyDarwinDockIcon для не-Cocoa backend и пустого embedded icon.
- [x] Проверен helper hasCustomIconFlag для отсутствующего файла, отсутствующего FinderInfo, нулевого FinderInfo, установленного kHasCustomIcon и другого флага.
- [x] Тест ограничен Darwin, где доступны FinderInfo xattr и реализация Dock icon.

## Инвариант

Пользовательский Finder icon определяется только битом kHasCustomIcon в байтах 8–9 FinderInfo; неподдерживаемые backend и отсутствие embedded icon не должны запускать AppKit.