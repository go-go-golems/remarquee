package device

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-go-golems/remarquee/pkg/devicecapture"
	"github.com/go-go-golems/remarquee/pkg/deviceevents"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type serveSettings struct {
	BindAddr string
	Username string
	Password string
	Unsafe   bool
}

func NewServeCommand() *cobra.Command {
	s := &serveSettings{}
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve device capture endpoints",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(cmd, s)
		},
	}
	cmd.Flags().StringVar(&s.BindAddr, "bind", ":2718", "Bind address for the device capture server")
	cmd.Flags().StringVar(&s.Username, "username", "admin", "Basic auth username")
	cmd.Flags().StringVar(&s.Password, "password", "password", "Basic auth password")
	cmd.Flags().BoolVar(&s.Unsafe, "unsafe", false, "Disable authentication (not recommended)")
	return cmd
}

func runServe(cmd *cobra.Command, s *serveSettings) error {
	reader, err := devicecapture.NewFramebufferReader()
	if err != nil {
		return err
	}
	defer func() {
		if err := reader.Close(); err != nil {
			log.Printf("warning: close framebuffer reader: %v", err)
		}
	}()

	var eventBus *deviceevents.PubSub
	if scanner, err := deviceevents.NewEventScanner(); err == nil {
		eventBus = deviceevents.NewPubSub()
		scanner.StartAndPublish(context.Background(), eventBus)
	} else {
		log.Printf("events disabled: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(reader.ScreenInfo()); err != nil {
			http.Error(w, "failed to encode info", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/api/v1/screenshot.raw", func(w http.ResponseWriter, r *http.Request) {
		info := reader.ScreenInfo()
		buf := make([]byte, info.ScreenSizeBytes)
		if err := reader.ReadFrame(buf); err != nil {
			http.Error(w, "failed to read framebuffer", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("X-Device-Model", info.Model)
		w.Header().Set("X-Screen-Width", fmt.Sprintf("%d", info.Width))
		w.Header().Set("X-Screen-Height", fmt.Sprintf("%d", info.Height))
		w.Header().Set("X-Bytes-Per-Pixel", fmt.Sprintf("%d", info.BytesPerPixel))
		if _, err := w.Write(buf); err != nil {
			http.Error(w, "failed to write response", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/api/v1/screenshot.png", func(w http.ResponseWriter, r *http.Request) {
		info := reader.ScreenInfo()
		buf := make([]byte, info.ScreenSizeBytes)
		if err := reader.ReadFrame(buf); err != nil {
			http.Error(w, "failed to read framebuffer", http.StatusInternalServerError)
			return
		}
		img, err := devicecapture.RawToRGBA(buf, info)
		if err != nil {
			http.Error(w, "failed to convert framebuffer", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		if err := devicecapture.EncodePNG(w, img); err != nil {
			http.Error(w, "failed to encode png", http.StatusInternalServerError)
			return
		}
	})

	mux.Handle("/api/v1/stream", newStreamHandler(reader))
	mux.Handle("/api/v1/events", newEventsHandler(eventBus))
	mux.Handle("/api/v1/gestures", newGesturesHandler(eventBus))

	handler := http.Handler(mux)
	if !s.Unsafe {
		handler = basicAuthMiddleware(handler, s.Username, s.Password)
	}

	server := &http.Server{
		Addr:    s.BindAddr,
		Handler: handler,
	}

	fmt.Fprintf(cmd.OutOrStdout(), "listening on %s\n", s.BindAddr)
	if err := server.ListenAndServe(); err != nil {
		return errors.Wrap(err, "device server failed")
	}
	return nil
}
