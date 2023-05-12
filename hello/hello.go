package hello

import (
	"context"
	"crypto/x509"
	_ "embed"
	"log"
	"os"

	"github.com/jackc/pgx/v4"
)

/*
//go:embed root.pem
var rootCert []byte
*/

var secrets struct {
	Connstring string //postgres
	RootCert   string //convert to byte
}

// Welcome to Encore!
// This is a simple "Hello World" project to get you started.
//
// To run it, execute "encore run" in your favorite shell.

// ==================================================================

// This is a simple REST API that responds with a personalized greeting.
// To call it, run in your terminal:
//
//	curl http://localhost:4000/hello/World
//
//encore:api public path=/hello/:name
func World(ctx context.Context, name string) (*Response, error) {
	//log.Println(string(rootCert))
	rootCAs := x509.NewCertPool()

	// Append the embedded root certificate to the certificate pool
	//if !rootCAs.AppendCertsFromPEM(rootCert)) {
	if !rootCAs.AppendCertsFromPEM([]byte(secrets.RootCert)) {
		log.Fatalf("Failed to append root certificate to the pool")
	}

	//connstring := os.Getenv("connstring")
	connstring := secrets.Connstring

	// Attempt to connect
	config, err := pgx.ParseConfig(os.ExpandEnv(connstring))
	if err != nil {
		log.Fatal("error configuring the database: ", err)
	}

	//you download root.crt from your cockroachlabs account
	//root.crt is at $HOME/postgresql
	//openssl x509 -in root.crt -out root.pem

	config.TLSConfig.RootCAs = rootCAs

	conn, err := pgx.ConnectConfig(context.Background(), config)
	if err != nil {
		log.Fatal("error connecting to the database: ", err)
	}
	defer conn.Close(context.Background())
	log.Println("Hey! You successfully connected to your CockroachDB cluster.")

	msg := "Hello, " + name + ". You successfully connected to your CockroachDB cluster!"
	return &Response{Message: msg}, nil
}

type Response struct {
	Message string
}
