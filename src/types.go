package src

type NetTotals struct {
	DailyIn    float64 `json:"daily_in"`
	DailyOut   float64 `json:"daily_out"`
	MonthlyIn  float64 `json:"monthly_in"`
	MonthlyOut float64 `json:"monthly_out"`
}

type DailyStat struct {
	Date string  `json:"date"`
	In   float64 `json:"in"`
	Out  float64 `json:"out"`
}

type MonthlyStat struct {
	Month string  `json:"month"`
	In    float64 `json:"in"`
	Out   float64 `json:"out"`
}