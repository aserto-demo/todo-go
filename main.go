package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"

	"github.com/aserto-dev/aserto-go/client"
	aserto "github.com/aserto-dev/aserto-go/client/authorizer"

	"github.com/gorilla/mux"

	"todo-go/directory"
	"todo-go/server"
	"todo-go/store"
)

// JWTValidator is a middleware that validates JWT tokens against the JWKS issuer
func JWTValidator(jwksKeysURL string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			keys, err := jwk.Fetch(r.Context(), jwksKeysURL)
			authorizationHeader := r.Header.Get("Authorization")
			tokenBytes := []byte(strings.Replace(authorizationHeader, "Bearer ", "", 1))

			jwt.WithVerifyAuto(nil)
			_, err = jwt.Parse(tokenBytes, jwt.WithKeySet(keys))

			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}

func main() {
	// Load environment variables
	if envFileError := godotenv.Load(); envFileError != nil {
		log.Fatal("Error loading .env file")
	}

	authorizerAddr := os.Getenv("ASERTO_AUTHORIZER_ADDRESS")

	if authorizerAddr == "" {
		authorizerAddr = "authorizer.prod.aserto.com:8443"
	}
	apiKey := os.Getenv("ASERTO_AUTHORIZER_API_KEY")
	tenantID := os.Getenv("ASERTO_TENANT_ID")
	jwksKeysUrl := os.Getenv("JWKS_URI")

	// Initialize the Aserto Client
	ctx := context.Background()
	asertoClient, asertoClientErr := aserto.New(
		ctx,
		client.WithAddr(authorizerAddr),
		client.WithTenantID(tenantID),
		client.WithAPIKeyAuth(apiKey),
	)

	if asertoClientErr != nil {
		log.Fatal("Failed to create authorizer client:", asertoClientErr)
	}

	// Initialize the Todo Store
	db, dbError := store.NewStore()
	if dbError != nil {
		log.Fatal("Failed to create store:", dbError)
	}

	// Initialize the Directory
	dir := directory.Directory{DirectoryClient: asertoClient.Directory}

	// Initialize the Server
	srv := server.Server{Store: db}

	// Set up routes
	router := mux.NewRouter()
	router.HandleFunc("/todos", srv.GetTodos).Methods("GET")
	router.HandleFunc("/todo", srv.InsertTodo).Methods("POST")
	router.HandleFunc("/todo/{ownerID}", srv.UpdateTodo).Methods("PUT")
	router.HandleFunc("/todo/{ownerID}", srv.DeleteTodo).Methods("DELETE")
	router.HandleFunc("/user/{userID}", dir.GetUser).Methods("GET")

	// Initialize the JWT Validator
	jwtValidator := JWTValidator(jwksKeysUrl)
	// Set up JWT validation middleware
	router.Use(jwtValidator)

	srv.Start(router)
}
