package main

import (
	"bufio"
	"errors"
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
)

type journalFollower struct {
	reader *sdjournal.JournalReader
	cancel chan time.Time
	done   chan struct{}
	io.Reader
}

// Close reader and free resources
func (j journalFollower) Close() error {
	close(j.cancel)
	<-j.done
	return j.reader.Close()
}

// Tail a unit journal
func Tail(unit string) (io.ReadCloser, error) {
	if !strings.HasSuffix(unit, ".service") {
		unit = unit + ".service"
	}
	log.Println("Tailing logs of unit ", unit)
	reader, err := sdjournal.NewJournalReader(sdjournal.JournalReaderConfig{
		NumFromTail: 1024,
		Matches: []sdjournal.Match{
			{Field: "_SYSTEMD_UNIT", Value: unit},
		},
	})
	if err != nil {
		return nil, err
	}
	r, w := io.Pipe()
	follower := journalFollower{
		cancel: make(chan time.Time),
		done:   make(chan struct{}),
		reader: reader,
		Reader: r,
	}
	go func() {
		defer close(follower.done)
		if err := reader.Follow(follower.cancel, w); err != nil {
			panic(err)
		}
	}()
	return follower, nil
}

// Config of the application
type Config struct {
	Unit string
	Port int
}

func config() (zero Config, err error) {

	flag.StringVar(&zero.Unit, "unit", "", "Systemd unit to tail")
	flag.IntVar(&zero.Port, "port", 8080, "HTTP listen at port")
	flag.Parse()

	zero.Unit = strings.TrimSpace(zero.Unit)
	if zero.Unit == "" {
		return zero, errors.New("Unit name cannot be empty")
	}
	if zero.Port <= 1024 || zero.Port >= 65536 {
		return zero, errors.New("Port must be between 1025 and 65535")
	}
	return zero, nil
}

// findLink scans reader for links and forwards then to chan
func findLink(r io.Reader, hits chan string) {
	scanner := bufio.NewScanner(r)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		line := scanner.Text()
		indx := strings.Index(line, "link.tidal.com")
		if indx >= 0 {
			link := line[indx:]
			if space := strings.Index(link, " "); space >= 0 {
				link = link[:space]
			}
			hits <- link
		}
	}
}

const Template = `
<html>
<head>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style type="text/css">
	.container {
		align: center;
		text-align: center;
		padding: 2rem;
		font-size: x-large;
		font-weight: bold;
	}
	form {
		display: inline-block;
	}
	</style>
</head>
<body>
	<div class="container">
		<a href="https://{{ .Link }}">{{ .Link }}</a>
		<form method="GET">
			<button>&#x21bb;</button>
		</form>
	</div>
</body>
</html>
`

// showLink builds http.Handler to show hits
func showLink(hits chan string) http.Handler {
	var link string
	tmpl := template.Must(template.New("index.html").Parse(Template))
	var m sync.Mutex
	go func() {
		for l := range hits {
			m.Lock()
			link = l
			m.Unlock()
			log.Println("Detected link ", l)
		}
	}()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.Lock()
		l := link
		m.Unlock()
		w.WriteHeader(http.StatusOK)
		if err := tmpl.Execute(w, struct{ Link string }{Link: l}); err != nil {
			panic(err)
		}
	})
}

func main() {

	conf, err := config()
	if err != nil {
		panic(err)
	}

	// Open a new "tailer" to follow journal
	tailer, err := Tail(conf.Unit)
	if err != nil {
		panic(err)
	}
	defer tailer.Close()

	// Open a link finder to keep track of latest link
	hits := make(chan string, 16)
	defer close(hits)
	go findLink(tailer, hits)

	// Serve http
	if err := http.ListenAndServe(":"+strconv.Itoa(conf.Port), showLink(hits)); err != nil {
		panic(err)
	}
}

