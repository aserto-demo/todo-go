package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/aserto-dev/aserto-go/authorizer/grpc"
	"github.com/aserto-dev/aserto-go/client"
	"github.com/aserto-dev/aserto-go/middleware"
	"github.com/aserto-dev/aserto-go/middleware/http/std"
	"github.com/gorilla/mux"

	"go-server/directory"
	"go-server/server"
	"go-server/store"
)

type Todo struct {
	ID        string `storm:"id"`
	Title     string
	Completed bool
	UserEmail string
	UserSub   string
}

func GetOwnerEmail(r io.Reader) (string, error) {
	todo := Todo{}
	err := json.NewDecoder(r).Decode(&todo)
	if err != nil {
		return "", errors.Wrap(err, "Failed decoding JSON ")
	}
	return todo.UserEmail, nil
}

func EmailResourceMapperr(r *http.Request) *structpb.Struct {
	defer r.Body.Close()

	email, err := GetOwnerEmail(r.Body)

	if err != nil {
		log.Println("Failed to get Owner Email:", err)
		return nil
	}

	v := map[string]interface{}{
		"ownerEmail": email,
	}

	resourceContext, err := structpb.NewStruct(v)
	if err != nil {
		log.Println(err)
		return nil
	}

	return resourceContext
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
	).WithResourceMapper(EmailResourceMapperr).WithPolicyFromURL(policyRoot).WithPolicyFromURL(policyRoot)

	mw.Identity = mw.Identity.JWT().FromHeader("Authorization")

	return mw, nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
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

	authzMiddleware, err := AsertoAuthorizer(authorizerAddr, tenantID, apiKey, policyID, policyRoot, decision)
	if err != nil {
		log.Fatal("Failed to create authorizer middleware:", err)
	}

	store.InitDB()
	router := mux.NewRouter()

	router.HandleFunc("/user/{sub}", directory.GetUser).Methods("GET")
	router.HandleFunc("/todos", server.GetTodos).Methods("GET")
	router.HandleFunc("/todo", server.InsertTodo).Methods("POST")
	router.HandleFunc("/todo", server.UpdateTodo).Methods("PUT")
	router.HandleFunc("/todo", server.DeleteTodo).Methods("DELETE")

	router.Use(authzMiddleware.Handler)

	server.Start(router)
}
