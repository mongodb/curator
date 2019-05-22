package gimlet

// NewBasicUser constructs a simple user. The underlying type has
// serialization tags.
func NewBasicUser(id, name, email, password, key string, roles []string, invalid bool) User {
	return &basicUser{
		ID:           id,
		Name:         name,
		EmailAddress: email,
		Password:     password,
		Key:          key,
		AccessRoles:  roles,
		Invalid:      invalid,
	}
}

// MakeBasicUser constructs an empty basic user structure to ease
// serialization.
func MakeBasicUser() User { return &basicUser{} }

type basicUser struct {
	ID           string   `bson:"_id" json:"id" yaml:"id"`
	Name         string   `bson:"name" json:"name" yaml:"name"`
	EmailAddress string   `bson:"email" json:"email" yaml:"email"`
	Password     string   `bson:"password" json:"password" yaml:"password"`
	Key          string   `bson:"key" json:"key" yaml:"key"`
	AccessRoles  []string `bson:"roles" json:"roles" yaml:"roles"`
	Invalid      bool     `bson:"invalid" json:"invalid" yaml:"invalid"`
}

func (u *basicUser) Username() string    { return u.ID }
func (u *basicUser) Email() string       { return u.EmailAddress }
func (u *basicUser) DisplayName() string { return u.Name }
func (u *basicUser) GetAPIKey() string   { return u.Key }
func (u *basicUser) Roles() []string {
	out := make([]string, len(u.AccessRoles))
	copy(out, u.AccessRoles)
	return out
}

// userHasRole determines if the user has the defined role.
func userHasRole(u User, role string) bool {
	for _, r := range u.Roles() {
		if r == role {
			return true
		}
	}

	return false
}

func userInSlice(u User, users []string) bool {
	if len(users) == 0 {
		return false
	}

	id := u.Username()
	for _, u := range users {
		if id == u {
			return true
		}
	}

	return false
}
