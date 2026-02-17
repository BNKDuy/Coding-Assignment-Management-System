package Types

type RequestCreateAssignment struct {
	Name      string   `json:"name"`
	Students  []string `json:"students"`
	Languages []string `json:"languages"`
}
