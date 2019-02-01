package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/benesch/unison-fsmonitor/fswatch"
	"github.com/benesch/unison-fsmonitor/pathtrie"
)

func parseCommand(msg string) (cmd string, args []string, err error) {
	var out []string
	for _, arg := range strings.Split(msg, " ") {
		arg, err := url.PathUnescape(arg)
		if err != nil {
			return "", nil, fmt.Errorf("parseCommand: %s", err)
		}
		out = append(out, arg)
	}
	if len(out) == 0 {
		return "", nil, errors.New("parseCommand: empty message")
	}
	log.Println("recv command", out[0], out[1:])
	return out[0], out[1:], nil
}

func sendCommand(cmd string, args ...string) {
	var msg strings.Builder
	msg.WriteString(cmd)
	for _, arg := range args {
		msg.WriteByte(' ')
		msg.WriteString(url.PathEscape(arg))
	}
	msg.WriteByte('\n')
	log.Println("send command", cmd, args)
	os.Stdout.WriteString(msg.String())
}

func scanLoop(commands chan<- string, errs chan<- error) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		commands <- scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		errs <- err
	}
	errs <- nil
}

var replicas = map[string]*replica{}

type replica struct {
	hash     string
	basePath string
	dirs     map[string]string
	changes  pathtrie.PathTrie
	waiting  bool
}

func run() error {
	fsWatcher := fswatch.New()

	messages := make(chan string)
	scanErr := make(chan error)
	go scanLoop(messages, scanErr)

	sendCommand("VERSION 1")
	if msg := <-messages; msg != "VERSION 1" {
		return fmt.Errorf("bad version handshake: %q", msg)
	}

	var state struct {
		r    *replica
		path string
	}
	for {
		select {
		case msg := <-messages:
			cmd, args, err := parseCommand(msg)
			if err != nil {
				return err
			}

			if cmd != "WAIT" {
				for _, r := range replicas {
					r.waiting = false
				}
			}

			switch cmd {
			case "DEBUG":
				if len(args) != 0 {
					return fmt.Errorf("expected no args for DEBUG, but got %d", len(args))
				}
				log.SetOutput(os.Stderr)

			case "START":
				if len(args) != 3 {
					return fmt.Errorf("expected three args for START, but got %d", len(args))
				} else if state.r != nil {
					return errors.New("START command issued with already-active replica")
				}
				hash, basePath, path := args[0], args[1], args[2]

				if r, ok := replicas[hash]; !ok {
					state.r = &replica{
						hash:     hash,
						basePath: basePath,
						dirs:     map[string]string{},
					}
					replicas[hash] = state.r
					state.r.dirs[basePath] = ""
					if err := fsWatcher.Watch(state.r.basePath); err != nil {
						return err
					}
				} else {
					state.r = r
				}
				state.path = path
				sendCommand("OK")

			case "DIR":
				if len(args) != 1 {
					return fmt.Errorf("expected one arg for DIR, but got %d", len(args))
				} else if state.r == nil {
					return errors.New("DIR command issued without active replica")
				}

				// Our watches are recursive, so we don't need to do anything
				// special when told about a child directory within a replica.
				sendCommand("OK")

			case "LINK":
				if len(args) != 1 {
					return fmt.Errorf("expected one arg for LINK, but got %d", len(args))
				} else if state.r == nil {
					return errors.New("LINK command issued without active replica")
				}

				relPath := filepath.Join(state.path, args[0])
				absPath := filepath.Join(state.r.basePath, relPath)
				if err := fsWatcher.Watch(absPath); err != nil {
					if os.IsNotExist(err) {
						// Broken symlink. This is unfortunate, because at any
						// point the symlink could become unbroken, e.g.,
						// because its target is created. In principle, we
						// could install a watcher for when the target is
						// created, but this seems to be more complicated than
						// it's worth.
						continue
					}
					return err
				}
				target, err := os.Readlink(absPath)
				if err != nil {
					return err
				}
				if !strings.HasSuffix(relPath, "/") {
					relPath += "/"
				}
				state.r.dirs[target] = relPath
				sendCommand("OK")

			case "DONE":
				if len(args) != 0 {
					return fmt.Errorf("expected no args for DONE, but got %d", len(args))
				} else if state.r == nil {
					return errors.New("DONE command issued without active replica")
				}

				state.r = nil
				state.path = ""

			case "WAIT":
				if len(args) != 1 {
					return fmt.Errorf("expected one arg for WAIT, but got %d", len(args))
				}
				hash := args[0]
				r, ok := replicas[hash]
				if !ok {
					return fmt.Errorf("unknown replica %q", hash)
				}

				if !r.changes.Empty() {
					sendCommand("CHANGES", hash)
				} else {
					r.waiting = true
				}

			case "CHANGES":
				if len(args) != 1 {
					return fmt.Errorf("expected one arg for CHANGES, but got %d", len(args))
				}
				hash := args[0]
				r, ok := replicas[hash]
				if !ok {
					return fmt.Errorf("unknown replica %q", hash)
				}

				r.changes.Walk(func(c string) {
					sendCommand("RECURSIVE", c)
				})
				r.changes.Clear()
				sendCommand("DONE")

			case "RESET":
				if len(args) != 1 {
					return fmt.Errorf("expected one arg for RESET, but got %d", len(args))
				}
				hash := args[0]
				r, ok := replicas[hash]
				if !ok {
					return fmt.Errorf("unknown replica %q", hash)
				}

				for _, d := range r.dirs {
					if err := fsWatcher.Unwatch(filepath.Join(r.basePath, d)); err != nil {
						return err
					}
				}
				delete(replicas, hash)

			default:
				return fmt.Errorf("unknown command %q", cmd)
			}

		case event := <-fsWatcher.Events:
			log.Print("recv filesystem event", event)
			var found bool
			for _, r := range replicas {
				for realPath, replPath := range r.dirs {
					if strings.HasPrefix(event.Path(), realPath) {
						path, err := filepath.Rel(realPath, event.Path())
						if err != nil {
							return fmt.Errorf("internal error: receiving filesystem event: %s", err)
						}
						path = filepath.Join(replPath, path)
						r.changes.Insert(path)
						if r.waiting {
							r.waiting = false
							sendCommand("CHANGES", r.hash)
						}
						found = true
					}
				}
			}
			if !found {
				return fmt.Errorf("filesystem event %s did not match any replica", event)
			}

		case err := <-scanErr:
			return err
		}
	}
}

func main() {
	log.SetPrefix("[unison-fsmonitor] ")
	log.SetOutput(ioutil.Discard)

	if err := run(); err != nil {
		sendCommand("ERROR", err.Error())
		os.Exit(1)
	}
}
