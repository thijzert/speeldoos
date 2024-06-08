package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/thijzert/go-resemble"

	esb "github.com/evanw/esbuild/pkg/api"
)

type job func(ctx context.Context) error

type compileConfig struct {
	Development    bool
	Quick          bool
	GOOS           string
	GOARCH         string
	PackageVersion string
}

func main() {
	if _, err := os.Stat("pkg/web/assets"); err != nil {
		log.Fatalf("Error: cannot find speeldoos assets directory. (error: %s)\nAre you running this from the repository root?", err)
	}

	var conf compileConfig
	watch := false
	run := false
	flag.BoolVar(&conf.Development, "development", false, "Create a development build")
	flag.BoolVar(&conf.Quick, "quick", false, "Create a development build")
	flag.StringVar(&conf.GOARCH, "GOARCH", "", "Cross-compile for architecture")
	flag.StringVar(&conf.GOOS, "GOOS", "", "Cross-compile for operating system")
	flag.StringVar(&conf.PackageVersion, "version", "", "Override embedded version number")
	flag.BoolVar(&watch, "watch", false, "Watch source tree for changes")
	flag.BoolVar(&run, "run", false, "Run speeldoos upon successful compilation")
	flag.Parse()

	if conf.Development && conf.Quick {
		log.Printf("")
		log.Printf("You requested a quick build. This will not re-build static assets.")
		log.Printf("")
	}

	var theJob job

	if run {
		theJob = func(ctx context.Context) error {
			err := compile(ctx, conf)
			if err != nil {
				return err
			}
			runArgs := append([]string{"./speeldoos"}, flag.Args()...)
			return passthru(ctx, runArgs...)
		}
	} else {
		theJob = func(ctx context.Context) error {
			return compile(ctx, conf)
		}
	}

	if watch {
		theJob = watchSourceTree([]string{"."}, []string{"*.go"}, theJob)
	}

	err := theJob(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}

func compileJavascript(target esb.Target, outDir string, entryPoints ...string) error {
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		return fmt.Errorf("cannot create output directory: %w", err)
	}
	result := esb.Build(esb.BuildOptions{
		EntryPoints: entryPoints,
		Target:      target,
		Outdir:      outDir,
		Write:       true,
	})

	var rv error
	for _, err := range result.Warnings {
		log.Printf("Warning: %s in %s on line %d", err.Text, err.Location.File, err.Location.Line)
	}
	for _, err := range result.Errors {
		log.Printf("Error: %s in %s on line %d", err.Text, err.Location.File, err.Location.Line)
		rv = fmt.Errorf("%s in %s on line %d", err.Text, err.Location.File, err.Location.Line)
	}

	return rv
}

func inlineSvg(w io.Writer, file string, fill string) error {
	buf, err := os.ReadFile(path.Join("pkg/web/assets/src/svg", file))
	if err != nil {
		return fmt.Errorf("error reading svg file: %w", err)
	}
	rv := string(buf)
	rv = strings.TrimSpace(rv)
	for len(rv) > 5 && (rv[:2] == "<?" || rv[:4] == "<!--") {
		search := "-->"
		if rv[:2] == "<?" {
			search = "?>"
		}
		idx := strings.Index(rv, search)
		if idx <= 0 {
			break
		}
		rv = strings.TrimSpace(rv[idx+len(search):])
	}

	if fill != "" {
		// Hack the fill colour in there.
		// (If it's stupid and works, it wasn't stupid.)
		idx := 1 + strings.Index(rv[1:], "<")
		if idx != -1 {
			idx1 := idx + strings.Index(rv[idx:], ">")
			idx2 := idx + strings.Index(rv[idx:], " ")
			if idx2 > -1 && idx2 < idx1 {
				idx1 = idx2
			}

			rv = rv[:idx1] + " fill=\"" + fill + "\"" + rv[idx1:]
		}
	}

	rv = strings.ReplaceAll(rv, "%", "%25")
	rv = strings.ReplaceAll(rv, "#", "%23")
	rv = strings.ReplaceAll(rv, "<", "%3C")
	rv = strings.ReplaceAll(rv, ">", "%3E")
	rv = strings.ReplaceAll(rv, "?", "%3F")
	rv = strings.ReplaceAll(rv, "\"", "'")
	rv = strings.ReplaceAll(rv, "\n", " ")

	_, err = fmt.Fprintf(w, "url(\"data:image/svg+xml,%s\")", rv)
	return err
}

func compileSCSS(ctx context.Context, conf compileConfig, file string) error {
	scssStyle := "compressed"
	scssMap := "--omit-map-comment"
	if conf.Development {
		scssStyle = "nested"
		scssMap = "--sourcemap"
	}

	if len(file) < 6 || file[0:1] == "_" || file[len(file)-5:] != ".scss" {
		return nil
	}
	sourceFile := path.Join("pkg/web/assets/src/scss", file)
	targetFile := path.Join("pkg/web/assets/dist/css", file[:len(file)-5]+".css")

	err := os.MkdirAll(path.Dir(targetFile), 0755)
	if err != nil {
		return fmt.Errorf("cannot create CSS output directory: %w", err)
	}
	err = passthru(ctx, "sassc", "--style", scssStyle, scssMap, sourceFile, targetFile)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("error compiling assets: error compiling '%s'", file))
	}

	// Inline SVG images
	if true {
		buf, err := os.ReadFile(targetFile)
		if err != nil {
			return fmt.Errorf("error reading css output: %w", err)
		}
		f, err := os.Create(targetFile)
		if err != nil {
			return fmt.Errorf("error reopening css output: %w", err)
		}
		defer f.Close()
		re := regexp.MustCompile("svg-load\\(\"([^\"]+)\"(,\\s*fill=([^\\)]+))?\\)")
		lastIndex := 0
		matches := re.FindAllSubmatchIndex(buf, -1)
		for _, m := range matches {
			_, err = f.Write(buf[lastIndex:m[0]])
			if err != nil {
				return fmt.Errorf("error writing css chunk: %w", err)
			}
			err = inlineSvg(f, string(buf[m[2]:m[3]]), string(buf[m[6]:m[7]]))
			if err != nil {
				return fmt.Errorf("error writing inlined svg image: %w", err)
			}
			lastIndex = m[1]
		}
		_, err = f.Write(buf[lastIndex:])
		if err != nil {
			return fmt.Errorf("error writing css chunk: %w", err)
		}
	}

	return nil
}

func compile(ctx context.Context, conf compileConfig) error {
	// Compile static assets: Javascript
	err := compileJavascript(esb.ESNext, "pkg/web/assets/dist/js", "pkg/web/assets/src/js/**/*.js")
	if err != nil {
		return fmt.Errorf("error compiling javascript: %w", err)
	}

	err = compileJavascript(esb.ES2015, "pkg/web/assets/dist/ancient-js", "pkg/web/assets/src/js/speeldoos.js")
	if err != nil {
		return fmt.Errorf("error compiling ancient javascript: %w", err)
	}

	// Compile static assets: SCSS
	if !conf.Development || !conf.Quick {
		err := compileSCSS(ctx, conf, "speeldoos.scss")
		if err != nil {
			return err
		}
		d, err := os.Open("pkg/web/assets/src/scss/pages")
		if err != nil {
			return err
		}
		pages, _ := d.Readdirnames(-1)
		for _, page := range pages {
			err := compileSCSS(ctx, conf, path.Join("pages", page))
			if err != nil {
				return err
			}
		}
	}

	// Embed static assets
	if err := os.Chdir("pkg/web/assets"); err != nil {
		return errors.Errorf("Error: cannot find speeldoos assets directory. (error: %s)\nAre you *sure* you're running this from the repository root?", err)
	}
	var emb resemble.Resemble
	emb.OutputFile = "../../../internal/web-plumbing/assets.go"
	emb.PackageName = "plumbing"
	emb.Debug = conf.Development
	emb.AssetPaths = []string{
		".",
	}
	if err := emb.Run(); err != nil {
		os.Chdir("../../..")
		return errors.WithMessage(err, "error running 'resemble'")
	}

	os.Chdir("../../..")

	if conf.PackageVersion == "" {
		gitDescCmd := exec.CommandContext(ctx, "git", "describe")
		gitDescribe, err := gitDescCmd.Output()
		if err != nil || len(gitDescribe) == 0 {
			conf.PackageVersion = "default version"
		} else {
			conf.PackageVersion = string(gitDescribe)
		}
	}

	// Build main executable
	execOutput := "speeldoos"
	if runtime.GOOS == "windows" || conf.GOOS == "windows" {
		execOutput = "speeldoos.exe"
	}

	gofiles, err := filepath.Glob("cmd/speeldoos/*.go")
	if err != nil || gofiles == nil {
		return errors.WithMessage(err, "error: cannot find any go files to compile.")
	}
	compileArgs := append([]string{
		"build",
		"-ldflags", "-X github.com/thijzert/speeldoos/pkg.PackageVersion=" + conf.PackageVersion,
		"-o", execOutput,
	}, gofiles...)

	compileCmd := exec.CommandContext(ctx, "go", compileArgs...)

	compileCmd.Env = append(compileCmd.Env, os.Environ()...)
	if conf.GOOS != "" {
		compileCmd.Env = append(compileCmd.Env, "GOOS="+conf.GOOS)
	}
	if conf.GOARCH != "" {
		compileCmd.Env = append(compileCmd.Env, "GOARCH="+conf.GOARCH)
	}

	err = passthruCmd(compileCmd)
	if err != nil {
		return errors.WithMessage(err, "compilation failed")
	}

	if conf.Development && !conf.Quick {
		log.Printf("")
		log.Printf("Development build finished.")
		log.Printf("")
	} else {
		log.Printf("Compilation finished.")
	}

	return nil
}

func passthru(ctx context.Context, argv ...string) error {
	c := exec.CommandContext(ctx, argv[0], argv[1:]...)
	return passthruCmd(c)
}

func passthruCmd(c *exec.Cmd) error {
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}

func watchSourceTree(paths []string, fileFilter []string, childJob job) job {
	return func(ctx context.Context) error {
		var mu sync.Mutex
		for {
			lastHash := sourceTreeHash(paths, fileFilter)
			current := lastHash
			cctx, cancel := context.WithCancel(ctx)
			go func() {
				mu.Lock()
				err := childJob(cctx)
				if err != nil {
					log.Printf("child process: %s", err)
				}
				mu.Unlock()
			}()

			for lastHash == current {
				time.Sleep(250 * time.Millisecond)
				current = sourceTreeHash(paths, fileFilter)
			}

			log.Printf("Source change detected - rebuilding")
			cancel()
		}
	}
}

func sourceTreeHash(paths []string, fileFilter []string) string {
	h := sha1.New()
	for _, d := range paths {
		h.Write(directoryHash(0, d, fileFilter))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func directoryHash(level int, filePath string, fileFilter []string) []byte {
	h := sha1.New()
	h.Write([]byte(filePath))

	fi, err := os.Stat(filePath)
	if err != nil {
		return h.Sum(nil)
	}
	if fi.IsDir() {
		base := filepath.Base(filePath)
		if level > 0 {
			if base == ".git" || base == ".." || base == "node_modules" {
				return []byte{}
			}
		}
		// recurse
		var names []string
		f, err := os.Open(filePath)
		if err == nil {
			names, err = f.Readdirnames(-1)
		}
		if err == nil {
			for _, name := range names {
				if name == "" || name[0] == '.' {
					continue
				}
				h.Write(directoryHash(level+1, path.Join(filePath, name), fileFilter))
			}
		}
	} else {
		if fileFilter != nil {
			found := false
			for _, pattern := range fileFilter {
				if ok, _ := filepath.Match(pattern, filePath); ok {
					found = true
				} else if ok, _ := filepath.Match(pattern, filepath.Base(filePath)); ok {
					found = true
				}
			}
			if !found {
				return []byte{}
			}
		}
		f, err := os.Open(filePath)
		if err == nil {
			io.Copy(h, f)
			f.Close()
		}
	}
	return h.Sum(nil)
}
