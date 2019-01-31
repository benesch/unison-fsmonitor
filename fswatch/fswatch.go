// Package fswatch provides a filesystem watcher.
package fswatch

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rjeczalik/notify"
)

// An Event represents a change to a watched filesystem.
type Event = notify.EventInfo

// An FSWatcher manages several file system watch points.
//
// Watch points can be added and removed during the lifetime of the FSWatcher.
// Events from all watch points are multiplexed over the Events channel.
type FSWatcher struct {
	Events chan Event

	chans map[string]chan Event
}

// New creates a new FSWatcher.
func New() *FSWatcher {
	return &FSWatcher{
		Events: make(chan Event),
		chans:  map[string]chan Event{},
	}
}

// Watch installs a new watch point at path. Any changes to path will cause
// events to be sent over the Events channel. If path specifies a directory,
// the directory and its children will be watched recursively.
//
// If path itself is a symlink, it is followed before the watch point is
// installed. Symlinks within path, however, are not followed.
func (w *FSWatcher) Watch(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		// Instruct our underlying notify watcher to create a recursive watch.
		path = filepath.Join(path, "...")
	}
	if _, ok := w.chans[path]; !ok {
		ch := make(chan Event, 1<<10)
		if err := notify.Watch(path, ch, notify.All); err != nil {
			return err
		}
		go func() {
			for ev := range ch {
				w.Events <- ev
			}
		}()
		w.chans[path] = make(chan Event)
	}
	return nil
}

// Unwatch removes an existing watch point. The watch point must have been
// previously registered with Watch with an identical path.
func (w *FSWatcher) Unwatch(path string) error {
	ch, ok := w.chans[path]
	if !ok {
		return fmt.Errorf("FSWatcher.Unwatch: no watcher registered for %q", path)
	}
	notify.Stop(ch)
	close(ch)
	delete(w.chans, path)
	return nil
}
