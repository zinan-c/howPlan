package models

type PlanSummary struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	StartDate  string `json:"startDate"`
	EndDate    string `json:"endDate"`
	CoverImage string `json:"coverImage"`
	IsSimple   bool   `json:"isSimple"`
}

type PlansIndex struct {
	Plans []PlanSummary `json:"plans"`
}

type PlanDetail struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	StartDate  string    `json:"startDate"`
	EndDate    string    `json:"endDate"`
	CoverImage string    `json:"coverImage"`
	IsSimple   bool      `json:"isSimple"`
	DayPlans   []DayPlan `json:"dayPlans"`
}

type DayPlan struct {
	DayNumber int         `json:"dayNumber"`
	Title     string      `json:"title"`
	Stops     []StopPoint `json:"stops"`
}

type StopPoint struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Latitude            float64  `json:"latitude"`
	Longitude           float64  `json:"longitude"`
	ActivityDescription string   `json:"activityDescription"`
	ImageURLs           []string `json:"imageUrls"`
}
