package Types

type User struct {
	username    string
	password    string
	role        Role
	assignments map[string]string
}

func NewUser(username string, password string, role Role) *User {
	return &User{
		username:    username,
		password:    password,
		role:        role,
		assignments: make(map[string]string),
	}
}

// Getters
func (u *User) GetUsername() string               { return u.username }
func (u *User) GetPassword() string               { return u.password }
func (u *User) GetRole() Role                     { return u.role }
func (u *User) GetAssignments() map[string]string { return u.assignments }

func (u *User) AddAssignment(id string, name string) { u.assignments[id] = name }

// Object comparison
func (u1 *User) IsEqual(u2 User) bool { return u1.username == u2.username }
