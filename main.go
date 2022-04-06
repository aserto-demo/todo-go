package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/aserto-dev/aserto-go/client"
	authz "github.com/aserto-dev/aserto-go/client/authorizer"
	"github.com/aserto-dev/aserto-go/middleware"
	"github.com/aserto-dev/aserto-go/middleware/http/std"
	"github.com/aserto-dev/go-grpc-authz/aserto/authorizer/authorizer/v1"
	"github.com/gorilla/mux"

	"todo-go/directory"
	"todo-go/server"
	"todo-go/store"
	"todo-go/structs"
)

type Todo = structs.Todo

func AsertoAuthorizer(authClient authorizer.AuthorizerClient, policyID, policyRoot, decision string) *std.Middleware {

	mw := std.New(
		authClient,
		middleware.Policy{
			ID:       policyID,
			Decision: decision,
		},
	)

	mw.Identity.JWT().FromHeader("Authorization")

	mw.WithResourceMapper(
		func(r *http.Request) *structpb.Struct {

			var todo Todo
			bodyBytes, _ := io.ReadAll(r.Body)

			r.Body.Close() //  must close
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			if err := json.Unmarshal(bodyBytes, &todo); err != nil {
				return nil
			}

			v := map[string]interface{}{
				"ownerID": todo.OwnerID,
			}

			resourceContext, err := structpb.NewStruct(v)
			if err != nil {
				log.Println(err)
			}
			return resourceContext
		},
	)

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

	// Initialize the Authorizer Client
	ctx := context.Background()
	authClient, authorizerClientErr := authz.New(
		ctx,
		client.WithAddr(authorizerAddr),
		client.WithTenantID(tenantID),
		client.WithAPIKeyAuth(apiKey),
	)

	if authorizerClientErr != nil {
		log.Fatal("Failed to create authorizer client:", authorizerClientErr)
	}

	// Initialize the Authorizer
	asertoAuthorizer := AsertoAuthorizer(authClient.Authorizer, policyID, policyRoot, decision)

	// Initialize the Todo Store
	db, dbError := store.NewStore()
	if dbError != nil {
		log.Fatal("Failed to create store:", dbError)
	}

	// Initialize the Directory
	dir := directory.Directory{DirectoryClient: authClient.Directory}

	// Initialize the Server
	srv := server.Server{Store: db}

	// Set up routes
	router := mux.NewRouter()
	router.HandleFunc("/user/{userID}", dir.GetUser).Methods("GET")
	router.HandleFunc("/todos", srv.GetTodos).Methods("GET")
	router.HandleFunc("/todo", srv.InsertTodo).Methods("POST")
	router.HandleFunc("/todo", srv.UpdateTodo).Methods("PUT")
	router.HandleFunc("/todo", srv.DeleteTodo).Methods("DELETE")

	// Set up middleware
	// log.Println(asertoAuthorizer)
	router.Use(asertoAuthorizer.Handler)

	srv.Start(router)
}
