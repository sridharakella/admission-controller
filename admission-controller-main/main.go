package main

import (
    "crypto/tls"
    "flag"
    "fmt"
    "log"
    "net/http"
)

var (
    tlsCertFile string
    tlsKeyFile  string
    port        int
)

func init() {
    flag.StringVar(&tlsCertFile, "tls-cert", "/etc/webhook/certs/tls.crt", "TLS certificate file")
    flag.StringVar(&tlsKeyFile, "tls-key", "/etc/webhook/certs/tls.key", "TLS key file")
    flag.IntVar(&port, "port", 8443, "HTTPS port")
    flag.Parse()
}

func main() {
    log.Println("Starting admission controller...")

    // Initialize the webhook handler
    webhookHandler := &WebhookHandler{}

    // Setup HTTP routes
    mux := http.NewServeMux()
    mux.HandleFunc("/mutate", webhookHandler.ServeMutate)
    mux.HandleFunc("/health", healthHandler)

    // Configure TLS
    cert, err := tls.LoadX509KeyPair(tlsCertFile, tlsKeyFile)
    if err != nil {
        log.Fatalf("Failed to load TLS certificates: %v", err)
    }

    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        MinVersion:   tls.VersionTLS12,
    }

    server := &http.Server{
        Addr:      fmt.Sprintf(":%d", port),
        TLSConfig: tlsConfig,
        Handler:   mux,
    }

    log.Printf("Server listening on port %d...", port)
    if err := server.ListenAndServeTLS("", ""); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}
