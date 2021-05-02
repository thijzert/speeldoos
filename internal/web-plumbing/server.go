package plumbing

import (
	"context"
	"html/template"
	"net/http"

	"github.com/thijzert/speeldoos/lib/wavreader/chunker"
	speeldoos "github.com/thijzert/speeldoos/pkg"
	"github.com/thijzert/speeldoos/pkg/web"
)

// A ServerConfig combines common options for running a HTTP frontend
type ServerConfig struct {
	Context      context.Context
	Library      *speeldoos.Library
	StreamConfig chunker.MP3ChunkConfig
}

// A Server wraps a HTTP frontend
type Server struct {
	context         context.Context
	config          ServerConfig
	mux             *http.ServeMux
	scheduler       *speeldoos.Scheduler
	chunker         chunker.Chunker
	parsedTemplates map[string]*template.Template
	nowPlaying      speeldoos.Performance
}

// New instantiates a new server instance
func New(config ServerConfig) (*Server, error) {
	s := &Server{
		context: config.Context,
		config:  config,
		mux:     http.NewServeMux(),
	}

	err := s.initAudioStream()
	if err != nil {
		return nil, err
	}

	s.mux.Handle("/", s.HTMLFunc(web.HomeHandler, "full/home"))
	s.mux.Handle("/status", s.HTMLFunc(web.StatusHandler, "full/status"))
	s.mux.Handle("/library", s.HTMLFunc(web.LibraryHandler, "full/library"))

	s.mux.Handle("/api/status/buffers", s.JSONFunc(web.BufferStatusHandler))
	s.mux.Handle("/api/search", s.HTMLFunc(web.SearchResultHandler, "fragment/searchResult"))
	s.mux.Handle("/api/queue/add", s.JSONFunc(web.AddQueueHandler))

	s.mux.Handle("/stream.mp3", s.JSONFunc(web.MP3StreamHandler))
	s.mux.Handle("/stream.wav", s.JSONFunc(web.WAVStreamHandler))
	s.mux.Handle("/now-playing", s.HTMLFunc(web.NowPlayingHandler, "fragment/nowPlaying"))

	s.mux.HandleFunc("/assets/", s.serveStaticAsset)

	return s, nil
}

// Close frees any held resources
func (s *Server) Close() error {
	// TODO: actually close some resources
	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) getState() web.State {
	rv := web.State{
		Library:   s.config.Library,
		RawStream: s.scheduler.AudioStream,
		MP3Stream: s.chunker,
	}
	if st, ok := s.chunker.(chunker.Statuser); ok {
		rv.Buffers.MP3Stream = st
	}
	if st, ok := s.scheduler.AudioStream.(chunker.Statuser); ok {
		rv.Buffers.Scheduler = st
	}
	if ad, err := s.scheduler.AudioStream.GetAssociatedData(); err == nil {
		if pf, ok := ad.(speeldoos.Performance); ok {
			rv.NowPlaying = pf
		}
	}

	s.scheduler.QueueMutex.RLock()
	rv.PlayQueue = append(rv.PlayQueue, s.scheduler.PlayQueue...)
	s.scheduler.QueueMutex.RUnlock()

	return rv
}

// setState writes back any modified fields to the global state
func (s *Server) setState(state web.State) error {
	if state.PlayQueueDirty {
		s.scheduler.QueueMutex.Lock()
		s.scheduler.PlayQueue = append(s.scheduler.PlayQueue[:0], state.PlayQueue...)
		s.scheduler.QueueMutex.Unlock()
	}

	return nil
}
