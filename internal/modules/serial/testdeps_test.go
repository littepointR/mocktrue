package serial

import (
	"github.com/littepointR/mocktrue/internal/core/config"
	"github.com/littepointR/mocktrue/internal/core/eventbus"
	"github.com/littepointR/mocktrue/internal/core/logging"
	"github.com/littepointR/mocktrue/internal/core/module"
)

// testDeps returns a minimal, non-nil Deps for lifecycle tests. It uses a
// discard logger and defaulted config; no filesystem or Wails needed.
func testDeps() module.Deps {
	return module.Deps{
		Bus:    eventbus.New(),
		Config: config.Default(),
		Logger: logging.Default(),
	}
}
