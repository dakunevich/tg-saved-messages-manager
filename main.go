package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"telegram-manager/internal/server"
	"telegram-manager/internal/tg"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// Initialize Telegram Client
	tgClient, err := tg.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Telegram client: %v. Make sure TG_APP_ID and TG_APP_HASH are set.", err)
	}

	// The client needs to connect and authenticate before we can really use it.
	// We'll run the client in a blocking manner as intended by gotd, but since we also need an HTTP server,
	// we will start the HTTP server inside the `onReady` callback or in a goroutine.
	// gotd's Run() blocks until connection is closed.

	// Better approach: Start HTTP server in a goroutine, but it needs a ready client.
	// Or: Use Client.StartAndListen which blocks. Inside the callback (when ready), we start the HTTP server?
	// No, `http.ListenAndServe` blocks too.

	// We will start the HTTP server in a goroutine, but it might fail requests until TG is ready.
	// Or we wait for TG to be ready.

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	err = tgClient.StartAndListen(ctx, func(ctx context.Context) error {
		// This callback is called when Auth is successful and client is ready.

		srv := server.NewServer(tgClient)
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}

		// Start HTTP Server blocking (it will listen on ctx.Done())
		// Sinc we are inside StartAndListen, blocking here would block the TG client loop if StartAndListen expects prompt return?
		// StartAndListen says: "It executes the 'onReady' callback when the client is authenticated and ready to query."
		// If StartAndListen logic (in client.go) loops `client.Run` which executes callback...
		// In gotd, `client.Run` blocks until the callback returns?
		// No, `client.Run` runs the callback once authenticated and keeps connection open *while* callback is running?
		// Let's check `client.go`.
		// `err := client.Run(ctx, func(ctx) error { ... return onReady(ctx) })`
		// So `client.Run` waits for `onReady` to return?
		// If `onReady` returns `nil`, `client.Run` returns `nil` (and closes connection)?
		// Yes, typically `client.Run` connects, auths, runs callback, then disconnects.
		// So we MUST block in `onReady` if we want to keep using the client!

		// The previous implementation had `<-ctx.Done()` at the end of callback.
		// So my new `srv.Start(ctx, port)` which blocks until ctx.Done is perfect.

		if err := srv.Start(ctx, port); err != nil {
			// If server error (not shutdown), we log it
			if err != context.Canceled {
				log.Printf("HTTP Server stopped with error: %v", err)
			}
		}

		// Ensure we respect context cancellation if Start returns early for some other reason
		// But Start blocks on ctx.Done.

		return nil
	})

	if err != nil {
		log.Fatalf("Telegram Client Error: %v", err)
	}
}
