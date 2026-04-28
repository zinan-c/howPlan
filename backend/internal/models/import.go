package models

type ImportPlanResponse struct {
	PlanID   string   `json:"planId"`
	Name     string   `json:"name"`
	Warnings []string `json:"warnings"`
}
