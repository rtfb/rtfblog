package rtfblog

import (
	"path/filepath"

	"github.com/nicksnyder/go-i18n/i18n"
	"github.com/rtfb/rtfblog/src/assets"
)

const (
	l10n = "l10n"
)

var (
	L10n i18n.TranslateFunc
)

func loadLanguage(assets *assets.AssetBin, langFile string) {
	fp := filepath.Join(l10n, langFile)
	if err := i18n.ParseTranslationFileBytes(fp, assets.MustLoad(fp)); err != nil {
		panic("Can't load language '" + langFile + "'; " + err.Error())
	}
}

// Loads translation files and inits L10n func that retrieves the translations.
// l10nDir is a name of a directory with translations.
// userLocale specifies a locale preferred by the user (a preference or accept
// header or language cookie).
func InitL10n(assets *assets.AssetBin, userLocale string) {
	loadLanguage(assets, "en-US.all.json")
	loadLanguage(assets, "lt-LT.all.json")
	defaultLocale := "en-US" // known valid locale
	L10n = i18n.MustTfunc(userLocale, defaultLocale)
	addTemplateFunc("L10n", L10n)
}
