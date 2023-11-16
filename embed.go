package embedded

import "embed"

//go:embed build/default.db build/static l10n tmpl
var Assets embed.FS
