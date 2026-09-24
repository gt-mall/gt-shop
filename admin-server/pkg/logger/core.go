package logger

import (
	"sync"

	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// switchableCore keeps loggers returned by L and S bound to the current output.
// Its lock prevents writes to a file writer while it is being closed.
type switchableCore struct {
	state  *coreState
	fields []zapcore.Field
}

type coreState struct {
	mu      sync.RWMutex
	core    zapcore.Core
	writers []*lumberjack.Logger
}

func newSwitchableCore(core zapcore.Core) *switchableCore {
	return &switchableCore{state: &coreState{core: core}}
}

func (c *switchableCore) Enabled(level zapcore.Level) bool {
	c.state.mu.RLock()
	defer c.state.mu.RUnlock()
	return c.state.core.Enabled(level)
}

func (c *switchableCore) With(fields []zapcore.Field) zapcore.Core {
	all := make([]zapcore.Field, 0, len(c.fields)+len(fields))
	all = append(all, c.fields...)
	all = append(all, fields...)
	return &switchableCore{state: c.state, fields: all}
}

func (c *switchableCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

// levelFilterCore preserves each destination's level check when a tee is
// written through the switchable core.
type levelFilterCore struct{ zapcore.Core }

func (c levelFilterCore) With(fields []zapcore.Field) zapcore.Core {
	return levelFilterCore{Core: c.Core.With(fields)}
}

func (c levelFilterCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	if !c.Enabled(entry.Level) {
		return nil
	}
	return c.Core.Write(entry, fields)
}

func (c *switchableCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	c.state.mu.RLock()
	defer c.state.mu.RUnlock()
	core := c.state.core
	if len(c.fields) > 0 {
		core = core.With(c.fields)
	}
	return core.Write(entry, fields)
}

func (c *switchableCore) Sync() error {
	c.state.mu.RLock()
	defer c.state.mu.RUnlock()
	return c.state.core.Sync()
}

func (c *switchableCore) swap(core zapcore.Core, writers []*lumberjack.Logger) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	c.state.core = core
	for _, writer := range c.state.writers {
		_ = writer.Close()
	}
	c.state.writers = writers
}
