package directory

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/aserto-dev/go-grpc/aserto/authorizer/directory/v1"
	"github.com/gorilla/mux"
)

type Directory struct {
	DirectoryClient directory.DirectoryClient
}

func (d *Directory) resolveUserID(ctx context.Context, sub string) (string, error) {
	idResponse, err := d.DirectoryClient.GetIdentity(ctx,
		&directory.GetIdentityRequest{Identity: sub},
	)

	return idResponse.GetId(), err
}

func (d *Directory) resolveUserByUserID(ctx context.Context, userID string) (*directory.GetUserResponse, error) {
	userResponse, err := d.DirectoryClient.GetUser(ctx,
		&directory.GetUserRequest{Id: userID},
	)

	return userResponse, err
}

func (d *Directory) resolveUser(ctx context.Context, sub string) (*directory.GetUserResponse, error) {
	userID, err := d.resolveUserID(ctx, sub)
	if err != nil {
		return nil, err
	}
	userResponse, err := d.resolveUserByUserID(ctx, userID)

	return userResponse, err

}

func (d *Directory) GetUser(w http.ResponseWriter, r *http.Request) {
	sub := mux.Vars(r)["sub"]

	user, err := d.resolveUser(r.Context(), sub)
	if err != nil {
		log.Fatal("Failed to resolve users:", err)
	}

	w.Header().Add("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(user.Result)
	if err != nil {
		log.Fatal("Failed to decode users:", err)
	}
}
