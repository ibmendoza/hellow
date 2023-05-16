package hello

import (
	"context"
	"crypto/x509"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	//"encore.dev/middleware"
	"encore.dev/rlog"

	"github.com/bokwoon95/sq"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/pgx/v4/stdlib"
	_ "github.com/mattn/go-sqlite3"
	"go4.org/syncutil"
	"golang.org/x/crypto/bcrypt"
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
	svcdb  *sql.DB
	router *mux.Router
}

/*
//encore:middleware global target=all
func CsrfMiddleware(req middleware.Request, next middleware.Next) middleware.Response {
	// If the payload has a Validate method, use it to validate the request.
	payload := req.Data().Payload
	if validator, ok := payload.(interface{ Validate() error }); ok {
		if err := validator.Validate(); err != nil {
			// If the validation fails, return an InvalidArgument error.
			err = errs.WrapCode(err, errs.InvalidArgument, "validation failed")
			return middleware.Response{Err: err}
		}
	}
	return next(req)

}
*/

func initService() (*Service, error) {
	// Write your service initialization code here.

	/*
		r := mux.NewRouter()
		csrfMiddleware := csrf.Protect([]byte("32-byte-long-auth-key"))

		api := r.PathPrefix("/api").Subrouter()
		api.Use(csrfMiddleware)
		api.HandleFunc("/user/{id}", GetUser).Methods("GET")

		http.ListenAndServe(":8000", r)
	*/

	svc := &Service{}

	svc.router = mux.NewRouter()
	//r := mux.NewRouter()
	csrfMiddleware := csrf.Protect([]byte("32-byte-long-auth-key"))
	api := svc.router.PathPrefix("/api").Subrouter()
	api.Use(csrfMiddleware)
	api.HandleFunc("/user/{id}", GetUser).Methods("GET")
	http.ListenAndServe(":8000", svc.router)

	err := svc.setup3(context.Background())
	if err != nil {
		log.Println("Error in initService, ", err)
		return nil, err
	}
	return svc, nil
}

/*
//encore:service
type Service struct {
	oldRouter *gin.Engine // existing HTTP router
}

// Route all requests to the existing HTTP router if no other endpoint matches.
//
//encore:api public raw path=/!fallback
func (s *Service) Fallback(w http.ResponseWriter, req *http.Request) {
	s.oldRouter.ServeHTTP(w, req)
}
*/

//encore:api public raw path=/!fallback
func (s *Service) Fallback(w http.ResponseWriter, req *http.Request) {
	//s.oldRouter.ServeHTTP(w, req)
	s.router.ServeHTTP(w, req)
}

// encor :api public raw method=POST path=/api/user
func GetUser(w http.ResponseWriter, r *http.Request) {
	// Authenticate the request, get the id from the route params,
	// and fetch the user from the DB, etc.

	// Get the token and pass it in the CSRF header. Our JSON-speaking client
	// or JavaScript framework can now read the header and return the token in
	// in its own "X-CSRF-Token" request header on the subsequent POST.
	var user Usr
	w.Header().Set("X-CSRF-Token", csrf.Token(r))
	b, err := json.Marshal(user)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write(b)
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

func (s *Service) authenticate(username, password string) (bool, error) {
	var hashedPassword string
	err := s.svcdb.QueryRowContext(context.Background(), "SELECT password FROM users WHERE username = ?", username).Scan(&hashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return false, nil
	}

	return true, nil
}

type LoginForm struct {
	Username string `validate:"required"`
	Password string `validate:"required"`
}

//encore:api public raw method=POST path=/login
func (s *Service) Login(w http.ResponseWriter, r *http.Request) {

	var form LoginForm
	if err := r.ParseForm(); err != nil {
		rlog.Error("Login ParseForm", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	//log.Println(r.Form)

	form.Username = r.Form.Get("username")
	form.Password = r.Form.Get("password")

	validate := validator.New()
	if err := validate.Struct(form); err != nil {
		rlog.Error("Login validate Struct form", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	success, err := s.authenticate(form.Username, form.Password)
	if err != nil {
		rlog.Error("Login authenticate", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if success {
		fmt.Fprintf(w, "Success")
	} else {
		fmt.Fprintf(w, "Fail")
	}

}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

//encore:api public raw method=POST path=/adduser
func (s *Service) AddUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var credentials Credentials
	err := decoder.Decode(&credentials)
	if err != nil {
		rlog.Error("AddUser credentials decode", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Hash the password
	hashedPassword, err := hashPassword(credentials.Password)
	if err != nil {
		rlog.Error("AddUser hashedPassword", err)
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Use pgx.Stdlib as the driver for sql.DB
	pgxConn, err := stdlib.AcquireConn(s.svcdb)
	if err != nil {
		rlog.Error("AddUser stdlib.AcquireConn", err)
		http.Error(w, "Failed to acquire connection", http.StatusInternalServerError)
		return
	}
	defer stdlib.ReleaseConn(s.svcdb, pgxConn)

	// Insert the credentials into the database
	// TODO: fix this
	sql := "insert into users(id, name, email, password, verified, created_at) values(2, $1, $2, $3, false, now())"
	_, err = pgxConn.Exec(context.Background(), sql, credentials.Username, "user1@example.com", hashedPassword)
	if err != nil {
		rlog.Error("AddUser insert users", err)
		http.Error(w, "Failed to insert credentials", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Credentials saved successfully")
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
