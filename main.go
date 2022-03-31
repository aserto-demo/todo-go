package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/aserto-dev/aserto-go/authorizer/grpc"
	"github.com/aserto-dev/aserto-go/client"
	authz "github.com/aserto-dev/aserto-go/client/authorizer"
	"github.com/aserto-dev/aserto-go/middleware"
	"github.com/aserto-dev/aserto-go/middleware/http/std"
	"github.com/gorilla/mux"

	"todo-go/directory"
	"todo-go/server"
	"todo-go/store"
	"todo-go/structs"
)

type Todo = structs.Todo

func GetOwnerEmail(r io.Reader) (string, error) {
	var todo Todo
	jsonErr := json.NewDecoder(r).Decode(&todo)
	if jsonErr != nil {
		return "", errors.New("Failed decoding JSON " + jsonErr.Error())
	}
	return todo.UserEmail, nil
}

func AsertoAuthorizer(addr, tenantID, apiKey, policyID, policyRoot, decision string) (*std.Middleware, error) {

	ctx := context.Background()
	authClient, err := grpc.New(
		ctx,
		client.WithAddr(addr),
		client.WithTenantID(tenantID),
		client.WithAPIKeyAuth(apiKey),
	)

	if err != nil {
		return nil, err
	}

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

			bodyBytes, _ := ioutil.ReadAll(r.Body)
			r.Body.Close() //  must close
			r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

			var email, getOwnerEmailError = GetOwnerEmail(bytes.NewReader(bodyBytes))

			if getOwnerEmailError != nil {
				log.Println("Failed to get Owner Email:", getOwnerEmailError)
			}

			v := map[string]interface{}{
				"ownerEmail": email,
			}

			resourceContext, err := structpb.NewStruct(v)
			if err != nil {
				log.Println(err)
			}
			return resourceContext
		},
	)

	mw.WithPolicyFromURL(policyRoot)
	return mw, nil

}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
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

	ctx := context.Background()
	authClient, err := authz.New(
			ctx,
			client.WithAddr("authorizer.prod.aserto.com:8443"),
			client.WithTenantID(tenantID),
			client.WithAPIKeyAuth(apiKey),
	)

	// Initialize the authorizer
	authorizer, err := AsertoAuthorizer(authorizerAddr, tenantID, apiKey, policyID, policyRoot, decision)
	if err != nil {
		log.Fatal("Failed to create authorizer:", err)
	}


	db, err := store.NewStore()
	if err != nil {
		log.Fatal("Failed to create store:", err)
	}

	dir := directory.Directory{AuthorizerClient: authClient, Context: ctx}
	srv := server.Server{Store: db}

	router := mux.NewRouter()

	router.HandleFunc("/user/{sub}", dir.GetUser).Methods("GET")
	router.HandleFunc("/todos", srv.GetTodos).Methods("GET")
	router.HandleFunc("/todo", srv.InsertTodo).Methods("POST")
	router.HandleFunc("/todo", srv.UpdateTodo).Methods("PUT")
	router.HandleFunc("/todo", srv.DeleteTodo).Methods("DELETE")

	router.Use(authorizer.Handler)

	srv.Start(router)
}
