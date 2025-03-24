package model

type Card struct {
	Schema string `json:"schema"`
	Config Config `json:"config"`
	Body   Body   `json:"body"`
	Header Header `json:"header"`
}

type Config struct {
	UpdateMulti bool `json:"update_multi"`
}

type Body struct {
	Direction string    `json:"direction"`
	Padding   string    `json:"padding"`
	Elements  []Element `json:"elements"`
}

type Element struct {
	Tag         string      `json:"tag"`
	Columns     []Column    `json:"columns"`
	Rows        []Row       `json:"rows"`
	RowHeight   string      `json:"row_height"`
	HeaderStyle HeaderStyle `json:"header_style"`
	PageSize    int         `json:"page_size"`
	Margin      string      `json:"margin"`
}

type Row struct {
	CustomerName  string `json:"customer_name"`
	CustomerScale string `json:"customer_scale"`
}

type Column struct {
	DataType        string `json:"data_type"`
	Name            string `json:"name"`
	DisplayName     string `json:"display_name"`
	HorizontalAlign string `json:"horizontal_align"`
	Width           string `json:"width"`
}

type HeaderStyle struct {
	BackgroundStyle string `json:"background_style"`
	Bold            bool   `json:"bold"`
	Lines           int    `json:"lines"`
}

type Header struct {
	Title    Title  `json:"title"`
	Subtitle Title  `json:"subtitle"`
	Template string `json:"template"`
	Padding  string `json:"padding"`
}

type Title struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

type FailedFiles struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Path    string `json:"path"`
}
