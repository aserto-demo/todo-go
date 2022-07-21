package directory

import (
	"context"
	"encoding/json"
	"net/http"

	"net/url"

	"github.com/aserto-dev/go-grpc/aserto/authorizer/directory/v1"
	"github.com/gorilla/mux"
)

type Directory struct {
	DirectoryClient directory.DirectoryClient
}

func (d *Directory) resolveUser(ctx context.Context, userID string) (*directory.GetUserResponse, error) {
	idResponse, getIdentityError := d.DirectoryClient.GetIdentity(ctx,
		&directory.GetIdentityRequest{Identity: userID},
	)

	if getIdentityError != nil {
		return nil, getIdentityError
	}

	userResponse, getUserError := d.DirectoryClient.GetUser(ctx,
		&directory.GetUserRequest{Id: idResponse.GetId()},
	)

	if getUserError != nil {
		return nil, getUserError
	}

	return userResponse, nil

}

func (d *Directory) GetUser(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userID"]
	_, err := url.QueryUnescape(userID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, resolveUserError := d.resolveUser(r.Context(), userID)
	if resolveUserError != nil {
		http.Error(w, resolveUserError.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	encodeJSONError := json.NewEncoder(w).Encode(user.Result)
	if encodeJSONError != nil {
		http.Error(w, encodeJSONError.Error(), http.StatusBadRequest)
		return
	}
}
