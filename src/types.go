package src

type NetTotals struct {
	DailyIn    float64 `json:"daily_in"`
	DailyOut   float64 `json:"daily_out"`
	MonthlyIn  float64 `json:"monthly_in"`
	MonthlyOut float64 `json:"monthly_out"`
}