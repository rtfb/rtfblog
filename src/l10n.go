package main

import (
	"path/filepath"

	"github.com/nicksnyder/go-i18n/i18n"
)

const (
	l10n = "l10n"
)

var (
	L10n i18n.TranslateFunc
)

// Loads translation files and inits L10n func that retrieves the translations.
// l10nDir is a name of a directory with translations.
// userLocale specifies a locale preferred by the user (a preference or accept
// header or language cookie).
func InitL10n(root, userLocale string) {
	l10nDir := filepath.Join(root, l10n)
	i18n.MustLoadTranslationFile(filepath.Join(l10nDir, "en-US.all.json"))
	i18n.MustLoadTranslationFile(filepath.Join(l10nDir, "lt-LT.all.json"))
	defaultLocale := "en-US" // known valid locale
	L10n = i18n.MustTfunc(userLocale, defaultLocale)
	AddTemplateFunc("L10n", L10n)
}
