package hello

import (
	"context"
	"crypto/x509"
	"database/sql"
	_ "embed"

	//"errors"
	//"fmt"
	"log"
	"os"

	/*
		"github.com/aws/aws-sdk-go-v2/config"
		"github.com/aws/aws-sdk-go-v2/service/s3"
		"github.com/aws/aws-sdk-go-v2/service/s3/types"
	*/
	"github.com/bokwoon95/sq"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/pgx/v4/stdlib"
	"go4.org/syncutil"
)

/*
//go:embed root.pem
var rootCert []byte
*/

type Usr struct {
	ID    int
	Name  string
	Email string
}

var secrets struct {
	Connstring string //postgres
	RootCert   string //convert to byte
}

var (
	// once is like sync.Once except it re-arms itself on failure
	once syncutil.Once
	// pool is the successfully created database connection pool,
	// or nil when no such pool has been setup yet.
	pool *pgxpool.Pool
	db   *sql.DB

	svc *Service
)

//encore:service
type Service struct {
	// Add your dependencies here
	svcdb *sql.DB
}

func initService() (*Service, error) {
	// Write your service initialization code here.
	svc := &Service{}
	err := svc.setup3(context.Background())
	if err != nil {
		log.Println("Error in initService, ", err)
		return nil, err
	}
	return svc, nil
}

func (s *Service) Shutdown(force context.Context) {
	s.svcdb.Close()
	db.Close()
	pool.Close()
}

// Get returns a database connection pool to the external database.
// It is lazily created on first use.
func Get(ctx context.Context) (*pgxpool.Pool, *sql.DB, error) {
	// Attempt to setup the database connection pool if it hasn't
	// already been successfully setup.
	err := once.Do(func() error {
		var err error
		pool, db, err = setup(ctx)
		return err
	})
	return pool, db, err
}

// setup attempts to set up a database connection pool.
func setup(ctx context.Context) (*pgxpool.Pool, *sql.DB, error) {
	//log.Println(string(rootCert))
	rootCAs := x509.NewCertPool()

	//log.Println(secrets.RootCert)

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

	_, err = pgx.ConnectConfig(context.Background(), config)
	if err != nil {
		log.Fatal("error connecting to the database: ", err)
	}
	//defer conn.Close(context.Background())
	pgpool, err := pgxpool.Connect(ctx, connstring)
	if err != nil {
		return nil, nil, err
	}

	//use std db, reuse secure config above
	connStr := stdlib.RegisterConnConfig(config)
	db, err = sql.Open("pgx", connStr)

	if err != nil {
		log.Println("Error opening db sql ", err)
		return nil, nil, err
	}

	return pgpool, db, nil
}

//encore:api public path=/hello2/:name
func World2(ctx context.Context, name string) (*Response, error) {
	var err error
	pool, db, err = Get(ctx)
	if err != nil {
		log.Println("World2 ", err)
		return nil, err
	}

	if db == nil {
		log.Println("error sql open std db ", err)
		return nil, err
	}

	var user Usr
	user, err = sq.FetchOne(db, sq.
		Queryf("SELECT {*} FROM users WHERE id = {}", 1).
		SetDialect(sq.DialectPostgres),
		func(row *sq.Row) Usr {
			return Usr{
				ID:    row.Int("id"),
				Name:  row.String("name"),
				Email: row.String("email"),
			}
		},
	)

	if err != nil {
		log.Println("Error in func World ,", err)
		return nil, err
	}

	log.Println(user.ID, " ", user.Name, " ", user.Email)

	msg := "Success sq connection"
	return &Response{Message: msg}, err
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

	log.Println(secrets.RootCert)

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

// setup attempts to set up a database connection pool.
func (s *Service) setup3(ctx context.Context) error {
	//log.Println(string(rootCert))
	rootCAs := x509.NewCertPool()

	//log.Println(secrets.RootCert)

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

	_, err = pgx.ConnectConfig(context.Background(), config)
	if err != nil {
		log.Fatal("error connecting to the database: ", err)
	}

	/*
		//defer conn.Close(context.Background())
		pgpool, err := pgxpool.Connect(ctx, connstring)
		if err != nil {
			return nil, nil, err
		}
	*/

	//use std db, reuse secure config above
	connStr := stdlib.RegisterConnConfig(config)
	s.svcdb, err = sql.Open("pgx", connStr)
	//db, err = sql.Open("pgx", connStr)

	if err != nil {
		log.Println("Error opening db sql ", err)
		return err
	}

	return nil
}

//encore:api public path=/hello3/:name
func (s *Service) World3(ctx context.Context, name string) (*Response, error) {
	var err error
	//pool, db, err = Get(ctx)  --no need to call Get in every API func
	if err != nil {
		log.Println("World2 ", err)
		return nil, err
	}

	if s.svcdb == nil {
		log.Println("error sql open std db ", err)
		return nil, err
	}

	var user Usr
	user, err = sq.FetchOne(s.svcdb, sq.
		Queryf("SELECT {*} FROM users WHERE id = {}", 1).
		SetDialect(sq.DialectPostgres),
		func(row *sq.Row) Usr {
			return Usr{
				ID:    row.Int("id"),
				Name:  row.String("name"),
				Email: row.String("email"),
			}
		},
	)

	if err != nil {
		log.Println("Error in func World ,", err)
		return nil, err
	}

	log.Println(user.ID, " ", user.Name, " ", user.Email)

	msg := "Success sq connection"
	return &Response{Message: msg}, err
}

/*
// LambdaHandler is the main AWS Lambda handler function
//
//encore:api public path=/lambda
func LambdaHandler(ctx context.Context) error {
	// Create a new S3 client
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"), // Specify the AWS region
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %v", err)
	}
	client := s3.NewFromConfig(cfg)

	// Define the S3 bucket and file information
	bucket := "lambda-s3" //"your-bucket-name"
	filename := "hello.txt"
	content := "Hello World"

	// Create the file locally
	file, err := os.Create("/tmp/" + filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Write content to the file
	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to file: %v", err)
	}

	// Check if the bucket already exists
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: &bucket,
	})
	if err != nil {
		var createBucketErr *types.BucketAlreadyOwnedByYou
		if !errors.As(err, &createBucketErr) {
			// Create the bucket if it doesn't exist
			_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
				Bucket: &bucket,
			})
			if err != nil {
				return fmt.Errorf("failed to create S3 bucket: %v", err)
			}
			log.Printf("S3 bucket created: %s\n", bucket)
		}
	}

	// Upload the file to S3
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &filename,
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %v", err)
	}

	log.Printf("File uploaded successfully to S3 bucket: %s, filename: %s\n", bucket, filename)
	return nil
}
*/
