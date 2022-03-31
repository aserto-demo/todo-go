package directory

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

type User struct {
	Id           string `json:"id"`
	Display_name string `json:"display_name"`
	Picture      string `json:"picture"`
	Email        string `json:"email"`
}

type UserResult struct {
	Result User `json:"result"`
}

type UserId struct {
	Id string `json:"id"`
}
type Sub struct {
	Sub string `json:"sub"`
}

func resolveUserId(authorizerServiceUrl string, tenantID string, apiKey string, sub string) (string, error) {
	client := &http.Client{}
	url := authorizerServiceUrl + "/api/v1/dir/identities"

	payloadBytes, err := json.Marshal(map[string]string{
		"identity": sub,
	})
	if err != nil {
		return "", errors.Wrap(err, "error marshalling identity")
	}
	payload := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", url, payload)

	if err != nil {
		return "", errors.Wrap(err, "error creating request")
	}

	req.Header.Add("aserto-tenant-id", tenantID)
	req.Header.Add("authorization", "basic "+apiKey)

	resp, err := client.Do(req)

	if err != nil {
		return "", errors.Wrap(err, "received an error from the directory")
	}

	userId := UserId{}
	err = json.NewDecoder(resp.Body).Decode(&userId)
	if err != nil {
		return "", errors.Wrap(err, "error decoding user id")
	}

	return userId.Id, nil
}

func resolveUserByUserId(authorizerServiceUrl string, tenantID string, apiKey string, userId string) (*User, error) {
	client := http.Client{}
	userResult := UserResult{}
	url := authorizerServiceUrl + "/api/v1/dir/users/" + userId + "?fields.mask=id,display_name,picture,email"

	req, requestError := http.NewRequest("GET", url, nil)

	if requestError != nil {
		return nil, requestError
	}

	req.Header.Add("aserto-tenant-id", tenantID)
	req.Header.Add("authorization", "basic "+apiKey)

	resp, responseError := client.Do(req)

	if responseError != nil {
		return nil, responseError
	}

	err := json.NewDecoder(resp.Body).Decode(&userResult)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding user")
	}

	return &userResult.Result, nil

}

func resolveUser(authorizerServiceUrl string, tenantID string, apiKey string, sub string) (*User, error) {
	userId, err := resolveUserId(authorizerServiceUrl, tenantID, apiKey, sub)
	if err != nil {
		return nil, err
	}
	return resolveUserByUserId(authorizerServiceUrl, tenantID, apiKey, userId)
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	sub := mux.Vars(r)["sub"]

	authorizerServiceUrl := os.Getenv("AUTHORIZER_SERVICE_ADDRESS")
	apiKey := os.Getenv("AUTHORIZER_API_KEY")
	tenantID := os.Getenv("TENANT_ID")

	user, err := resolveUser(authorizerServiceUrl, tenantID, apiKey, sub)
	if err != nil {
		log.Fatal("Failed to resolve users:", err)
	}

	w.Header().Add("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		log.Fatal("Failed to decode users:", err)
	}
}
