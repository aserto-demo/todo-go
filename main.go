package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/aserto-dev/aserto-go/client"
	authz "github.com/aserto-dev/aserto-go/client/authorizer"
	"github.com/aserto-dev/aserto-go/middleware"
	"github.com/aserto-dev/aserto-go/middleware/http/std"
	"github.com/aserto-dev/go-grpc-authz/aserto/authorizer/authorizer/v1"
	"github.com/gorilla/mux"

	"todo-go/directory"
	"todo-go/server"
	"todo-go/store"
)

func AsertoAuthorizer(authClient authorizer.AuthorizerClient, policyID, policyRoot, decision string) *std.Middleware {
	mw := std.New(
		authClient,
		middleware.Policy{
			ID:       policyID,
			Decision: decision,
		},
	)

	mw.Identity.JWT().FromHeader("Authorization")
	mw.WithPolicyFromURL(policyRoot)
	return mw
}

func main() {
	// Load environment variables
	if envFileError := godotenv.Load(); envFileError != nil {
		log.Fatal("Error loading .env file")
	}

	authorizerAddr := os.Getenv("AUTHORIZER_ADDRESS")

	if authorizerAddr == "" {
		authorizerAddr = "authorizer.prod.aserto.com:8443"
	}
	apiKey := os.Getenv("AUTHORIZER_API_KEY")
	policyID := os.Getenv("POLICY_ID")
	tenantID := os.Getenv("TENANT_ID")
	policyRoot := os.Getenv("POLICY_ROOT")
	decision := "allowed"

	// Initialize the Aserto Client
	ctx := context.Background()
	asertoClient, asertoClientErr := authz.New(
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

	// Initialize the Authorizer
	asertoAuthorizer := AsertoAuthorizer(asertoClient.Authorizer, policyID, policyRoot, decision)

	// Set up middleware
	router.Use(asertoAuthorizer.Handler)

	srv.Start(router)
}
