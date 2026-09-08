package main

import (
	"testing"

	"github.com/unxed/f4/internal/config"
)

// A fallback language may only fill keys the primary language lacks — it must
// never override the primary. With primary English the embedded base already
// covers every key, so a configured fallback must change nothing.
func TestInitLang_EnglishPrimaryNotOverriddenByFallback(t *testing.T) {
	oldLang, oldFallback := config.App.Language, config.App.FallbackLanguage
	defer func() {
		config.App.Language, config.App.FallbackLanguage = oldLang, oldFallback
		InitLang()
	}()

	config.App.Language = "en"
	config.App.FallbackLanguage = "ru"
	InitLang()

	if got := Msg("Menu.Exit"); got != "E&xit" {
		t.Errorf("primary=en fallback=ru: Msg(Menu.Exit) = %q, want English \"E&xit\"", got)
	}
}
