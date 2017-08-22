package hivemind

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Package hivemind implements a simple worker pool, basically a `sync.Waitgroup` on steroids. Jobs added to a hive mind are executed by workers until `Wait` is called.

type Hivemind struct {
	output  io.Writer
	workers []*worker
	inbox   chan Job
	outbox  chan changeEvent
	wg      sync.WaitGroup
	running bool
}

func New(workers int) *Hivemind {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	rv := &Hivemind{
		workers: make([]*worker, workers),
		wg:      sync.WaitGroup{},
		output:  os.Stdout,
	}

	return rv
}

func (h *Hivemind) init() {
	h.inbox = make(chan Job)
	h.outbox = make(chan changeEvent)

	for i, _ := range h.workers {
		h.workers[i] = &worker{
			ID:     i,
			Title:  "",
			Logger: log.New(h.output, fmt.Sprintf("[%2x] ", i), log.Ltime|log.Lmicroseconds),
			Inbox:  h.inbox,
			Outbox: h.outbox,
		}
		go h.workers[i].Work()
	}

	go h.listen()

	h.running = true
}

func (h *Hivemind) listen() {
	for ev := range h.outbox {
		if (ev.Flags & fTitle) != 0 {
			h.workers[ev.Sender].Title = ev.Title
		}
		if (ev.Flags & fLog) != 0 {
			h.writeString(ev.LogLine)
		}

		if (ev.Flags & (fLog | fTitle)) != 0 {
			r := "Current worker status:"
			for _, w := range h.workers {
				r += fmt.Sprintf(" [%s]", w.Title)
			}
			h.writeString(r + "\n")
		}
	}
}

func (h *Hivemind) writeString(s string) {
	h.uncaringWrite([]byte(s))
}

// Write bytes, retrying if the entire buffer couldn't be written at once, but ignoring all other errors
func (h *Hivemind) uncaringWrite(b []byte) {
	var err error = nil
	var n, i int = 0, 0
	for err == nil && n < len(b) {
		i, err = h.output.Write(b[n:])
		n += i
	}
}

// Add a job to the queue.
// If they weren't active already, spin up the workers
func (h *Hivemind) AddJob(j Job) {
	if !h.running {
		h.init()
	}
	h.inbox <- j
}

// Wait for all jobs to finish, and shut down the hive
func (h *Hivemind) Wait() {
	close(h.inbox)
	h.wg.Wait()
	close(h.outbox)
	h.running = false
}

// JC for 'Job Control'
type JC interface {
	SetTitle(string)
	Println(string)
	Printf(string, ...interface{})
}

type Job interface {
	Run(j JC) error
}

type eventFlags int

const (
	fTitle eventFlags = 1 << iota
	fLog
)

type changeEvent struct {
	Sender  int
	Flags   eventFlags
	Title   string
	LogLine string
}

type worker struct {
	ID     int
	Title  string
	Logger *log.Logger
	Inbox  chan Job
	Outbox chan changeEvent
}

func (w *worker) Work() {
	for j := range w.Inbox {
		err := j.Run(w)
		if err != nil {
			w.Logger.Printf("Error: %s", err)
		}
	}
}

func (w *worker) SetTitle(title string) {
	w.Outbox <- changeEvent{
		Sender: w.ID,
		Flags:  fTitle,
		Title:  title,
	}
}

func (w *worker) Println(ln string) {
	w.Printf(ln)
}

func (w *worker) Printf(format string, argc ...interface{}) {
	ln := fmt.Sprintf(format, argc...)
	for len(ln) > 0 && ln[len(ln)-1] == '\n' {
		ln = ln[0 : len(ln)-1]
	}
	if ln == "" {
		return
	}

	ln = fmt.Sprintf("[%2x] %s  %s", w.ID, time.Now().Format("2006-01-02 15:04:05.000"), ln)
	ln = strings.Replace(ln, "\n", "\n                         ", -1) + "\n"

	w.Outbox <- changeEvent{
		Sender:  w.ID,
		Flags:   fLog,
		LogLine: ln,
	}
}
