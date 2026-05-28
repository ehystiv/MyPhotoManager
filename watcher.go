package main

import (
	"context"
	"fmt"
	"io"
	"time"
)

// Watcher esegue scansioni periodiche della cartella input e organizza i nuovi file.
type Watcher struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// StartWatch avvia il watcher in background. La goroutine si ferma quando Stop() viene chiamato.
// organizePhotos è idempotente: i file già organizzati finiscono nelle managed folders e vengono
// esclusi automaticamente dalla collezione successiva.
func StartWatch(inputDir, outputDir string, opts OrganizerOptions, logW io.Writer, onProgress ProgressFunc, onStats func(OrganizerStats)) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	w := &Watcher{cancel: cancel, done: make(chan struct{})}

	go func() {
		defer close(w.done)

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		fmt.Fprintln(logW, "Watch attivo. Scansione automatica ogni 10 secondi…\n")

		for {
			select {
			case <-ctx.Done():
				fmt.Fprintln(logW, "\nWatch fermato.")
				return
			case t := <-ticker.C:
				fmt.Fprintf(logW, "[%s] Scansione…\n", t.Format("15:04:05"))
				stats, err := organizePhotos(ctx, inputDir, outputDir, opts, onProgress, logW)
				if err != nil && ctx.Err() == nil {
					fmt.Fprintf(logW, "Errore: %v\n", err)
				}
				if onStats != nil && ctx.Err() == nil {
					onStats(stats)
				}
			}
		}
	}()

	return w
}

// Stop termina il watcher e attende che la goroutine sia uscita.
func (w *Watcher) Stop() {
	w.cancel()
	<-w.done
}
