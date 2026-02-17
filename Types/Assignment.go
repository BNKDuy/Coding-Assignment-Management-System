package Types

import (
	"maps"

	"github.com/google/uuid"
)

type Assignment struct {
	id               string
	name             string
	owner            User
	path             string
	submissions      map[string]string
	allowedLanguages map[string]Runner
}

func NewAssignment(owner User, name string, path string, students []string, languages []string) *Assignment {
	submissions := make(map[string]string)
	for _, student := range students {
		submissions[student] = "No submission"
	}

	allowedLanguages := make(map[string]Runner)
	if len(languages) == 0 {
		maps.Copy(allowedLanguages, SupportedLanguage)
	} else {
		for _, v := range languages {
			runner, ok := SupportedLanguage[v]
			if !ok {
				continue
			}
			allowedLanguages[v] = runner
		}
	}

	return &Assignment{
		id:               uuid.New().String(),
		name:             name,
		owner:            owner,
		path:             path,
		submissions:      submissions,
		allowedLanguages: allowedLanguages,
	}
}

// Getters
func (a *Assignment) GetId() string                          { return a.id }
func (a *Assignment) GetName() string                        { return a.name }
func (a *Assignment) GetOwner() User                         { return a.owner }
func (a *Assignment) GetPath() string                        { return a.path }
func (a *Assignment) GetAllSubmissions() map[string]string   { return a.submissions }
func (a *Assignment) GetAllowedLanguages() map[string]Runner { return a.allowedLanguages }

func (a *Assignment) SetPath(path string)                    { a.path = path }
func (a *Assignment) SetGrade(username string, grade string) { a.submissions[username] = grade }

func (a *Assignment) HasPermission(username string) bool { _, ok := a.submissions[username]; return ok }
