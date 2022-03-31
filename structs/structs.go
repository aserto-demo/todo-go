package structs

type Todo struct {
	ID        string
	UserEmail string
	Title     string
	Completed bool
	UserSub   string
}

type User struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Picture     string `json:"picture"`
	Email       string `json:"email"`
}

type UserResult struct {
	Result User `json:"result"`
}

type UserID struct {
	ID string `json:"id"`
}
type Sub struct {
	Sub string `json:"sub"`
}
