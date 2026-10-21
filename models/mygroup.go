package models

type MyGroup struct {
	Group
	Priority string `json:"priority"`
}
