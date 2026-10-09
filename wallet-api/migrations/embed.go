// Package migrations SQL migration dosyalarını binary'ye gömer; sunucu
// MIGRATE_ON_START=true ile açılışta bunları uygular (goose).
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
