package tasks

type Task struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Meta     string `json:"meta"`
	Priority string `json:"priority"`
	Done     bool   `json:"done"`
}