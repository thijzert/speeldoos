package plumbing

import (
	"html/template"
	"net/http"

	"github.com/thijzert/speeldoos/lib/wavreader/chunker"
	speeldoos "github.com/thijzert/speeldoos/pkg"
	"github.com/thijzert/speeldoos/pkg/web"
)

// A ServerConfig combines common options for running a HTTP frontend
type ServerConfig struct {
	Library      *speeldoos.Library
	StreamConfig chunker.MP3ChunkConfig
}

// A Server wraps a HTTP frontend
type Server struct {
	config          ServerConfig
	mux             *http.ServeMux
	chunker         chunker.Chunker
	parsedTemplates map[string]*template.Template
	nowPlaying      speeldoos.Performance
}

// New instantiates a new server instance
func New(config ServerConfig) (*Server, error) {
	s := &Server{
		config: config,
		mux:    http.NewServeMux(),
	}

	err := s.initAudioStream()
	if err != nil {
		return nil, err
	}

	s.mux.Handle("/", s.HTMLFunc(web.HomeHandler, "full/home"))

	s.mux.Handle("/now-playing", s.HTMLFunc(web.NowPlayingHandler, "fragment/nowPlaying"))

	s.mux.HandleFunc("/assets/", s.serveStaticAsset)
	s.mux.HandleFunc("/stream.mp3", s.asyncStreamHandler)

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
	return web.State{
		Library:    s.config.Library,
		NowPlaying: s.nowPlaying,
	}
}

// setState writes back any modified fields to the global state
func (s *Server) setState(web.State) error {
	return nil
}
